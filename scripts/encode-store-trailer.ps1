[CmdletBinding()]
param([ValidateSet('EN','ES','Both')][string]$Language = 'Both')
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
$ffmpeg = Join-Path $repoRoot 'build\video-runtime\ffmpeg.exe'
$languages = if ($Language -eq 'Both') { @('EN','ES') } else { @($Language) }
foreach ($locale in $languages) {
    $directory = Join-Path $repoRoot "assets\store\$locale"
    $preview = Join-Path $directory 'aura-trailer-1080p-v1.mp4'
    $master = Join-Path $directory 'aura-trailer-store-1080p-v1.mp4'
    if (-not (Test-Path -LiteralPath $preview)) { throw "Missing edited trailer: $preview" }
    & $ffmpeg -hide_banner -loglevel error -y -i $preview -map 0:v:0 -map 0:a:0 `
        -c:v libx264 -preset medium -profile:v high -pix_fmt yuv420p -r 30 `
        -b:v 50M -minrate 50M -maxrate 50M -bufsize 100M `
        -g 15 -keyint_min 15 -sc_threshold 0 -bf 2 -flags +cgop -coder 1 `
        -x264-params 'nal-hrd=cbr:force-cfr=1:open-gop=0' -tag:v avc1 `
        -c:a aac -profile:a aac_low -b:a 384k -ar 48000 -ac 2 `
        -movflags +faststart -use_editlist 0 `
        -metadata 'title=Aura Player - Real app trailer' $master
    if ($LASTEXITCODE -ne 0) { throw "Store encoding failed: $locale" }
    & $ffmpeg -hide_banner -loglevel error -y -ss 1.5 -i $master -frames:v 1 (Join-Path $directory 'aura-trailer-thumbnail-1920x1080-v1.png')
    if ($LASTEXITCODE -ne 0) { throw "Thumbnail render failed: $locale" }
    Write-Output "Store trailer: $master"
}
