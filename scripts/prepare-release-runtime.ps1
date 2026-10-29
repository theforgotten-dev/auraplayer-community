[CmdletBinding()]
param()
$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$runtimeDir = Join-Path $repoRoot "build\release-runtime"
New-Item -ItemType Directory -Force -Path $runtimeDir | Out-Null
$archive = Join-Path $runtimeDir "libmpv-lgpl.zip"
$url = "https://github.com/mpv-player/mpv/releases/download/git-release/libmpv-v0.41.0-dev-gdc4b285c9-40753318951-x86_64-w64-mingw32-lgpl.zip"
$archiveHash = "6E1FD3FA52BE457058A8088A531C434E4354F5A203201F81567010D894A9E13D"
$dllHash = "21C6366F4BA77DA78411115E85A1815BCB00A2352FBE06A7020690DD47C182CF"
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
    variant="LGPL"; architecture="windows-amd64"; mpvCommit="dc4b285c9";
    archiveUrl=$url; archiveSHA256=$archiveHash; dllSHA256=$dllHash;
    sourceUrl="https://github.com/mpv-player/mpv/tree/dc4b285c9";
    buildWorkflow="https://github.com/mpv-player/mpv/blob/dc4b285c9/.github/workflows/build.yml"
} | ConvertTo-Json | Set-Content -Encoding UTF8 (Join-Path $runtimeDir "runtime-manifest.json")
Write-Output "Verified LGPL runtime: $dll"
