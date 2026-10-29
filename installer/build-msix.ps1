[CmdletBinding()]
param(
    [string]$Version = "1.2.3.0",
    [string]$PayloadVersion = "1.2.3",
    [string]$IdentityFile = (Join-Path $PSScriptRoot "store-identity.json")
)
$ErrorActionPreference = "Stop"
if ($Version -notmatch '^\d+\.\d+\.\d+\.0$') { throw "Store version must have four numeric parts and end in .0." }
$parsedVersion = [version]$Version
if ($parsedVersion.Major -lt 1 -or @($parsedVersion.Major,$parsedVersion.Minor,$parsedVersion.Build) | Where-Object { $_ -gt 65535 }) { throw "Invalid MSIX version." }
if ($PayloadVersion -notmatch '^\d+\.\d+\.\d+$') { throw "Invalid payload version." }
$repoRoot = Split-Path -Parent $PSScriptRoot
$payload = Join-Path $repoRoot "dist\package-$PayloadVersion"
$identity = Get-Content -LiteralPath $IdentityFile -Raw | ConvertFrom-Json
foreach ($field in @("Name","Publisher","PublisherDisplayName","DisplayName")) {
    if (-not $identity.$field) { throw "Missing Store identity field: $field" }
}
$sdkRoot = Join-Path ${env:ProgramFiles(x86)} "Windows Kits\10\bin"
$sdk = Get-ChildItem -LiteralPath $sdkRoot -Directory | Where-Object { $_.Name -match '^10\.0\.\d+\.0$' } | Sort-Object { [version]$_.Name } -Descending | Where-Object { Test-Path -LiteralPath (Join-Path $_.FullName 'x64\makeappx.exe') } | Select-Object -First 1
if (-not $sdk) { throw "Windows SDK MakeAppx.exe not found." }
$makeappx = Join-Path $sdk.FullName 'x64\makeappx.exe'
$workRoot = Join-Path $repoRoot "build\msix"
# A unique stage avoids deleting or mixing previous payloads.
$stage = Join-Path $workRoot ($Version + '-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $stage | Out-Null
foreach ($name in @('aura-native.exe','libmpv-2.dll','FORMATS.json','licenses')) {
    $source = Join-Path $payload $name
    if (-not (Test-Path -LiteralPath $source)) { throw "Missing release payload: $source. Build a release first." }
    Copy-Item -LiteralPath $source -Destination $stage -Recurse -Force
}
$productVersion = (Get-Item -LiteralPath (Join-Path $stage 'aura-native.exe')).VersionInfo.ProductVersion
if ($productVersion -ne $PayloadVersion) { throw "Payload version mismatch: $productVersion vs $PayloadVersion" }
Copy-Item -LiteralPath (Join-Path $repoRoot 'LICENSE') -Destination (Join-Path $stage 'LICENSE-Aura.txt')
Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'LEEME.txt') -Destination $stage
$legalDir = Join-Path $stage 'legal'
New-Item -ItemType Directory -Force -Path $legalDir | Out-Null
Copy-Item -Path (Join-Path $repoRoot 'legal\*.txt') -Destination $legalDir

Add-Type -AssemblyName System.Drawing
$assets = Join-Path $stage 'Assets'
New-Item -ItemType Directory -Force -Path $assets | Out-Null
$icon = [Drawing.Image]::FromFile((Join-Path $repoRoot 'assets\aura-icon.png'))
try {
    foreach ($size in @(44,50,150)) {
        $bitmap = New-Object Drawing.Bitmap($size,$size)
        $graphics = [Drawing.Graphics]::FromImage($bitmap)
        try {
            $graphics.Clear([Drawing.Color]::Transparent)
            $graphics.InterpolationMode = [Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
            $scale = [math]::Min($size / $icon.Width, $size / $icon.Height)
            $width = [int]($icon.Width * $scale); $height = [int]($icon.Height * $scale)
            $graphics.DrawImage($icon,[int](($size-$width)/2),[int](($size-$height)/2),$width,$height)
            $bitmap.Save((Join-Path $assets "Logo$size.png"),[Drawing.Imaging.ImageFormat]::Png)
        } finally { $graphics.Dispose(); $bitmap.Dispose() }
    }
} finally { $icon.Dispose() }

$name = [Security.SecurityElement]::Escape($identity.Name)
$publisher = [Security.SecurityElement]::Escape($identity.Publisher)
$publisherDisplay = [Security.SecurityElement]::Escape($identity.PublisherDisplayName)
$display = [Security.SecurityElement]::Escape($identity.DisplayName)
$manifest = @"
<?xml version="1.0" encoding="utf-8"?>
<Package xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10"
 xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"
 xmlns:rescap="http://schemas.microsoft.com/appx/manifest/foundation/windows10/restrictedcapabilities"
 IgnorableNamespaces="uap rescap">
 <Identity Name="$name" Publisher="$publisher" Version="$Version" ProcessorArchitecture="x64" />
 <Properties>
  <DisplayName>$display</DisplayName><PublisherDisplayName>$publisherDisplay</PublisherDisplayName>
  <Description>Reproductor de audio, video y subtitulos</Description><Logo>Assets\Logo50.png</Logo>
 </Properties>
 <Resources><Resource Language="es-CL" /><Resource Language="en-US" /></Resources>
 <Dependencies><TargetDeviceFamily Name="Windows.Desktop" MinVersion="10.0.17763.0" MaxVersionTested="10.0.26100.0" /></Dependencies>
 <Applications>
  <Application Id="Aura" Executable="aura-native.exe" EntryPoint="Windows.FullTrustApplication">
   <uap:VisualElements DisplayName="$display" Description="Reproductor de audio, video y subtitulos"
    Square150x150Logo="Assets\Logo150.png" Square44x44Logo="Assets\Logo44.png" BackgroundColor="transparent" />
  </Application>
 </Applications>
 <Capabilities><rescap:Capability Name="runFullTrust" /></Capabilities>
</Package>
"@
[IO.File]::WriteAllText((Join-Path $stage 'AppxManifest.xml'),$manifest,(New-Object Text.UTF8Encoding($false)))
$output = Join-Path $repoRoot "dist\Aura-Player-Store-$Version-x64.msix"
if (Test-Path -LiteralPath $output) { throw "Output already exists; choose a new version: $output" }
& $makeappx pack /d $stage /p $output /v
if ($LASTEXITCODE -ne 0) { throw "MSIX validation or packaging failed." }
$hash = Get-FileHash -LiteralPath $output -Algorithm SHA256
"$($hash.Hash.ToLower())  $(Split-Path -Leaf $output)" | Set-Content -Encoding ASCII ($output + '.sha256')
Write-Output "MSIX (unsigned, for Store submission): $output"
Write-Output "Stage: $stage"
