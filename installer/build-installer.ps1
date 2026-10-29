[CmdletBinding()]
param([string]$Version = "1.2.3")
$ErrorActionPreference = "Stop"
if ($Version -notmatch '^\d+\.\d+\.\d+$') { throw "Use a numeric version such as 1.1.0." }
$repoRoot = Split-Path -Parent $PSScriptRoot
$outputDir = Join-Path $repoRoot "dist"
$packageDir = Join-Path $outputDir "package-$Version"
$runtimeDir = Join-Path $repoRoot "build\release-runtime"
$executable = Join-Path $packageDir "aura-native.exe"
New-Item -ItemType Directory -Force -Path $packageDir | Out-Null
& (Join-Path $repoRoot "scripts\prepare-release-runtime.ps1")
& (Join-Path $repoRoot "scripts\prepare-windows-icon.ps1")
$previousGOOS = $env:GOOS
$previousGOARCH = $env:GOARCH
$env:GOOS = "windows"
$env:GOARCH = "amd64"
Push-Location $repoRoot
try {
    & go test ./...
    if ($LASTEXITCODE -ne 0) { throw "Tests failed." }
    & go vet ./...
    if ($LASTEXITCODE -ne 0) { throw "go vet failed." }
    & go run github.com/tc-hib/go-winres@v0.3.3 simply --arch amd64,arm64 --out rsrc --manifest gui --icon assets/aura-icon.png --product-name "Aura Player" --file-description "Aura Player - Audio y video" --original-filename "aura-native.exe" --product-version $Version --file-version $Version
    if ($LASTEXITCODE -ne 0) { throw "Windows resource generation failed." }
    Write-Output "Building standard Go executable (no obfuscation)."
    & go build -trimpath -ldflags "-H=windowsgui" -o $executable .
    if ($LASTEXITCODE -ne 0) { throw "Release build failed." }
    $formatData = & go run ./cmd/mediaformats
    if ($LASTEXITCODE -ne 0) { throw "Could not enumerate supported media formats." }
    [IO.File]::WriteAllText((Join-Path $packageDir "FORMATS.json"),($formatData -join [Environment]::NewLine))
    $formats = $formatData | ConvertFrom-Json
    $associationEntries = foreach ($extension in @($formats.Audio) + @($formats.Video)) {
        'Root: HKCU; Subkey: "Software\Classes\' + $extension + '\OpenWithProgids"; ValueType: string; ValueName: "AuraMusicPlayer.Audio"; ValueData: ""; Flags: uninsdeletevalue; Tasks: fileassoc'
    }
    $associationEntries | Set-Content -Encoding UTF8 (Join-Path $packageDir "file-associations.iss")
    Copy-Item -LiteralPath (Join-Path $runtimeDir "libmpv-2.dll") -Destination $packageDir
    Copy-Item -LiteralPath (Join-Path $repoRoot "LICENSE") -Destination (Join-Path $packageDir "LICENSE-Aura.txt")
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot "LEEME.txt") -Destination $packageDir
    $legalDir = Join-Path $packageDir "legal"
    New-Item -ItemType Directory -Force -Path $legalDir | Out-Null
    Copy-Item -Path (Join-Path $repoRoot "legal\*.txt") -Destination $legalDir -Force
    $licenseDir = Join-Path $packageDir "licenses"
    New-Item -ItemType Directory -Force -Path $licenseDir | Out-Null
    Copy-Item -LiteralPath (Join-Path $runtimeDir "LICENSE.LGPL") -Destination (Join-Path $licenseDir "libmpv-LGPL-2.1.txt")
    Copy-Item -LiteralPath (Join-Path $runtimeDir "LGPL-3.0.txt"),(Join-Path $runtimeDir "GPL-3.0.txt"),(Join-Path $runtimeDir "runtime-manifest.json") -Destination $licenseDir
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot "THIRD-PARTY-NOTICES.txt") -Destination $licenseDir
    $moduleInfo = & go list -m -f '{{.Path}}|{{.Version}}|{{.Dir}}' all
    if ($LASTEXITCODE -ne 0) { throw "Could not list dependency licenses." }
    $moduleInfo | ForEach-Object { $entry = $_.Split('|'); "$($entry[0]) $($entry[1])" } | Set-Content -Encoding UTF8 (Join-Path $licenseDir "go-modules.txt")
    foreach ($module in $moduleInfo) {
        $parts = $module.Split('|')
        if ($parts.Length -ne 3 -or -not $parts[2] -or $parts[0] -eq "playerdoit") { continue }
        $destination = Join-Path $licenseDir ($parts[0] -replace '[/\\:]','_')
        $licenseFiles = Get-ChildItem -LiteralPath $parts[2] -File | Where-Object { $_.Name -match '^(LICENSE|LICENCE|COPYING|NOTICE|AUTHORS)' }
        if ($licenseFiles) {
            New-Item -ItemType Directory -Force -Path $destination | Out-Null
            $licenseFiles | Copy-Item -Destination $destination -Force
        }
    }
}
finally {
    Pop-Location
    $env:GOOS = $previousGOOS
    $env:GOARCH = $previousGOARCH
}
$compilerCommand = Get-Command ISCC.exe -ErrorAction SilentlyContinue
$compiler = if ($compilerCommand) { $compilerCommand.Source } else { $null }
if (-not $compiler) {
    $locations = @(
        (Join-Path ${env:ProgramFiles(x86)} "Inno Setup 6\ISCC.exe"),
        (Join-Path $env:ProgramFiles "Inno Setup 6\ISCC.exe"),
        (Join-Path $env:LOCALAPPDATA "Programs\Inno Setup 6\ISCC.exe")
    )
    $compiler = $locations | Where-Object { Test-Path -LiteralPath $_ } | Select-Object -First 1
}
if (-not $compiler) { throw "Inno Setup 6 was not found." }
& $compiler "/DAppVersion=$Version" "/DSourceDir=$packageDir" "/O$outputDir" (Join-Path $PSScriptRoot "aura.iss")
if ($LASTEXITCODE -ne 0) { throw "Inno Setup compilation failed." }
$installer = Join-Path $outputDir "Aura-Player-Setup-$Version.exe"
if (-not (Test-Path -LiteralPath $installer)) { throw "Installer was not created." }
$portableArchive = Join-Path $outputDir "Aura-Player-Portable-$Version.zip"
$portableFiles = @("aura-native.exe","libmpv-2.dll","LEEME.txt","LICENSE-Aura.txt","FORMATS.json","licenses","legal") | ForEach-Object { Join-Path $packageDir $_ }
Compress-Archive -Path $portableFiles -DestinationPath $portableArchive -Force
Get-FileHash -LiteralPath $installer,$portableArchive -Algorithm SHA256 | ForEach-Object {
    "$($_.Hash.ToLower())  $(Split-Path -Leaf $_.Path)"
} | Set-Content -Encoding ASCII (Join-Path $outputDir "Aura-Player-$Version-SHA256.txt")
Write-Output "Installer: $installer"
Write-Output "Portable: $portableArchive"
