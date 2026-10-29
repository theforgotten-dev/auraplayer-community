[CmdletBinding()]
param()
$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$runtimeDir = Join-Path $repoRoot "build\release-runtime"
New-Item -ItemType Directory -Force -Path $runtimeDir | Out-Null
$archive = Join-Path $runtimeDir "libmpv-lgpl.zip"
$url = "https://github.com/mpv-player/mpv/releases/download/git-release/libmpv-v0.41.0-dev-g703e59b0e-40117195812-x86_64-w64-mingw32-lgpl.zip"
$archiveHash = "4C9D0E3FF428CB12483D0D27736A17D7E14D161024052F526E90F447EF20B452"
$dllHash = "812165F8A581AD4CD21CD5C9547F2F5E6EE2C6D7511268E27CD60EA1F5D85CD1"
if (-not (Test-Path -LiteralPath $archive)) { Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $archive }
if ((Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash -ne $archiveHash) { throw "LGPL archive checksum mismatch." }
$dll = Join-Path $runtimeDir "libmpv-2.dll"
if (-not (Test-Path -LiteralPath $dll) -or (Get-FileHash -LiteralPath $dll -Algorithm SHA256).Hash -ne $dllHash) {
    & tar -xf $archive -C $runtimeDir
    if ($LASTEXITCODE -ne 0) { throw "Could not extract LGPL runtime." }
}
if ((Get-FileHash -LiteralPath $dll -Algorithm SHA256).Hash -ne $dllHash) { throw "LGPL DLL checksum mismatch." }
foreach ($license in @(@("LGPL-3.0.txt","https://www.gnu.org/licenses/lgpl-3.0.txt"),@("GPL-3.0.txt","https://www.gnu.org/licenses/gpl-3.0.txt"))) {
    $destination = Join-Path $runtimeDir $license[0]
    if (-not (Test-Path -LiteralPath $destination)) { Invoke-WebRequest -UseBasicParsing -Uri $license[1] -OutFile $destination }
}
@{
    variant="LGPL"; architecture="windows-amd64"; mpvCommit="703e59b0e";
    archiveUrl=$url; archiveSHA256=$archiveHash; dllSHA256=$dllHash;
    sourceUrl="https://github.com/mpv-player/mpv/tree/703e59b0e";
    buildWorkflow="https://github.com/mpv-player/mpv/blob/703e59b0e/.github/workflows/build.yml"
} | ConvertTo-Json | Set-Content -Encoding UTF8 (Join-Path $runtimeDir "runtime-manifest.json")
Write-Output "Verified LGPL runtime: $dll"
