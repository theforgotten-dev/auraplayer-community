[CmdletBinding()]
param([string]$OutputDirectory = "")

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
if (-not $OutputDirectory) {
    $OutputDirectory = Join-Path $repoRoot "build\video-runtime"
}
New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
$OutputDirectory = (Resolve-Path -LiteralPath $OutputDirectory).Path
$archive = Join-Path $OutputDirectory "mpv-dev.7z"
$url = "https://github.com/shinchiro/mpv-winbuild-cmake/releases/download/20261021/mpv-dev-x86_64-20261021-git-e471496c56.7z"
$expectedHash = "EFC719EFF3DC5954B4A5EFC800632808BCEC01CC9034940DD14E5EB7FE9EA633"

if (-not (Test-Path -LiteralPath $archive)) {
    Invoke-WebRequest -Uri $url -OutFile $archive
}
$actualHash = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash
if ($actualHash -ne $expectedHash) {
    throw "Video runtime archive checksum mismatch. Expected $expectedHash, got $actualHash."
}
$dllPath = Join-Path $OutputDirectory "libmpv-2.dll"
$dllHash = "04E4D27DC385DC18DDD93FBE69C75E635B864F3C6EC53A9122DF54C2F7842F4A"
$runtimeReady = (Test-Path -LiteralPath $dllPath) -and ((Get-FileHash -LiteralPath $dllPath -Algorithm SHA256).Hash -eq $dllHash)
if (-not $runtimeReady) {
    & tar -xf $archive -C $OutputDirectory libmpv-2.dll
    if ($LASTEXITCODE -ne 0) { throw "Could not extract libmpv-2.dll using tar. Close Aura before replacing its runtime." }
    if ((Get-FileHash -LiteralPath $dllPath -Algorithm SHA256).Hash -ne $dllHash) {
        throw "Extracted video runtime checksum mismatch."
    }
}
Write-Output "Local Windows x64 video runtime ready: $OutputDirectory\libmpv-2.dll"
Write-Output "Build Aura into this directory, or set AURA_MPV_DLL to the absolute DLL path."
