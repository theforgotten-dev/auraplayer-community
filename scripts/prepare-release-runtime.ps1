[CmdletBinding()]
param()
$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$runtimeDir = Join-Path $repoRoot "build\release-runtime"
New-Item -ItemType Directory -Force -Path $runtimeDir | Out-Null
$archive = Join-Path $runtimeDir "libmpv-lgpl.zip"
$url = "https://github.com/mpv-player/mpv/releases/download/git-release/libmpv-v0.41.0-dev-g8e5a96fc4-40997856045-x86_64-w64-mingw32-lgpl.zip"
$archiveHash = "4F69A8412C1CED9E1EC328C6E1C2E8CE4701FE2DB8D7D2CC6F4C45A1525F1963"
$dllHash = "547D5C3558116BD628DC2CA3202D00B3D377A08A84732692415B2624C2765063"
if (-not (Test-Path -LiteralPath $archive)) { Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $archive }
if ((Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash -ne $archiveHash) { throw "LGPL archive checksum mismatch." }
$dll = Join-Path $runtimeDir "libmpv-2.dll"
if (-not (Test-Path -LiteralPath $dll) -or (Get-FileHash -LiteralPath $dll -Algorithm SHA256).Hash -ne $dllHash) {
    & tar -xf $archive -C $runtimeDir
    if ($LASTEXITCODE -ne 0) { throw "Could not extract LGPL runtime." }
}
if ((Get-FileHash -LiteralPath $dll -Algorithm SHA256).Hash -ne $dllHash) { throw "LGPL DLL checksum mismatch." }
foreach ($license in @("LGPL-3.0.txt","GPL-3.0.txt")) {
    Copy-Item -LiteralPath (Join-Path $repoRoot "legal/licenses/$license") -Destination (Join-Path $runtimeDir $license) -Force
}
@{
    variant="LGPL"; architecture="windows-amd64"; mpvCommit="8e5a96fc4";
    archiveUrl=$url; archiveSHA256=$archiveHash; dllSHA256=$dllHash;
    sourceUrl="https://github.com/mpv-player/mpv/tree/8e5a96fc4";
    buildWorkflow="https://github.com/mpv-player/mpv/blob/8e5a96fc4/.github/workflows/build.yml"
} | ConvertTo-Json | Set-Content -Encoding UTF8 (Join-Path $runtimeDir "runtime-manifest.json")
Write-Output "Verified LGPL runtime: $dll"
