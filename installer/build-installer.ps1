[CmdletBinding()]
param(
    [string]$Version = "1.0.0"
)

$ErrorActionPreference = "Stop"

if ($Version -notmatch '^\d+\.\d+\.\d+([+-][0-9A-Za-z.-]+)?$') {
    throw "Version must use semantic version format, for example 1.2.3."
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$outputDir = Join-Path $repoRoot "dist"
$packageDir = Join-Path $outputDir "package"
$executable = Join-Path $packageDir "aura-native.exe"
$portableArchive = Join-Path $outputDir "Aura-Music-Player-Portable-$Version.zip"
$scriptPath = Join-Path $PSScriptRoot "aura.iss"

New-Item -ItemType Directory -Force -Path $packageDir | Out-Null
New-Item -ItemType Directory -Force -Path $outputDir | Out-Null

$env:GOOS = "windows"
$env:GOARCH = "amd64"

Push-Location $repoRoot
try {
    & go build -trimpath -ldflags "-H=windowsgui -s -w" -o $executable .
    if ($LASTEXITCODE -ne 0) {
        throw "Go build failed with exit code $LASTEXITCODE."
    }
}
finally {
    Pop-Location
}

$compilerCommand = Get-Command "ISCC.exe" -ErrorAction SilentlyContinue
$compiler = if ($compilerCommand) { $compilerCommand.Source } else { $null }

if (-not $compiler) {
    $installLocations = @(
        (Join-Path ${env:ProgramFiles(x86)} "Inno Setup 6\ISCC.exe"),
        (Join-Path $env:ProgramFiles "Inno Setup 6\ISCC.exe"),
        (Join-Path $env:LOCALAPPDATA "Programs\Inno Setup 6\ISCC.exe")
    )
    $compiler = $installLocations | Where-Object { Test-Path $_ } | Select-Object -First 1
}

if (-not $compiler) {
    throw "Inno Setup 6 was not found. Install it from https://jrsoftware.org/isinfo.php and rerun this script."
}

& $compiler "/DAppVersion=$Version" "/DSourceDir=$packageDir" "/O$outputDir" $scriptPath
if ($LASTEXITCODE -ne 0) {
    throw "Inno Setup compilation failed with exit code $LASTEXITCODE."
}

$installer = Join-Path $outputDir "Aura-Music-Player-Setup-$Version.exe"
if (-not (Test-Path $installer)) {
    throw "Inno Setup completed, but the expected installer was not created: $installer"
}

Compress-Archive -Path (Join-Path $packageDir "*") -DestinationPath $portableArchive -Force
if (-not (Test-Path $portableArchive)) {
    throw "Portable package was not created: $portableArchive"
}

Write-Output "Portable package created: $portableArchive"
Write-Output "Installer created: $installer"
