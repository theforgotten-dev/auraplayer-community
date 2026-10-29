[CmdletBinding()]
param([string]$Version = "1.2.3")
$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
& (Join-Path $repoRoot "scripts\prepare-windows-icon.ps1")
$package = Join-Path $repoRoot "dist\package-$Version"
$testRoot = Join-Path $repoRoot "build\installer-smoke"
$installDir = Join-Path $testRoot "installed"
New-Item -ItemType Directory -Force -Path $testRoot | Out-Null
$compiler = Join-Path $env:LOCALAPPDATA "Programs\Inno Setup 6\ISCC.exe"
if (-not (Test-Path -LiteralPath $compiler)) {
    $compiler = (Get-Command ISCC.exe -ErrorAction Stop).Source
}
# Identical payload and installer sections, with an isolated uninstall identity
# and no compression. This does not replace an existing Aura installation.
& $compiler "/DSmokeTest=1" "/DAppVersion=$Version" "/DSourceDir=$package" "/O$testRoot" "/FAura-Player-Smoke" (Join-Path $PSScriptRoot "aura.iss") | Set-Content (Join-Path $testRoot "compile.log")
if ($LASTEXITCODE -ne 0) { throw "Smoke installer compilation failed." }
$setup = Join-Path $testRoot "Aura-Player-Smoke.exe"
$arguments = @('/VERYSILENT','/SUPPRESSMSGBOXES','/NORESTART','/NOICONS','/ACCEPTEULA=2026-10-24.2','/TASKS=""',('/DIR="' + $installDir + '"'),('/LOG="' + (Join-Path $testRoot "install.log") + '"'))
$install = Start-Process -FilePath $setup -ArgumentList $arguments -WindowStyle Hidden -Wait -PassThru
if ($install.ExitCode -ne 0) { throw "Test installation failed: $($install.ExitCode)." }
foreach ($file in @("aura-native.exe","libmpv-2.dll","LEEME.txt","LICENSE-Aura.txt","licenses\runtime-manifest.json","legal\EULA.es.txt","legal\EULA.en.txt","legal\USO-RESPONSABLE.txt")) {
    $installed = Join-Path $installDir $file
    if (-not (Test-Path -LiteralPath $installed)) { throw "Missing installed file: $file" }
    if ((Get-FileHash -LiteralPath $installed).Hash -ne (Get-FileHash -LiteralPath (Join-Path $package $file)).Hash) { throw "Installed file checksum mismatch: $file" }
}
Write-Output "Installation verified: $installDir"
Write-Output "Uninstaller: $(Join-Path $installDir 'unins000.exe')"
