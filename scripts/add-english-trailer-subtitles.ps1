[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
$ffmpeg = Join-Path $repoRoot 'build\video-runtime\ffmpeg.exe'
$outDir = Join-Path $repoRoot 'assets\store\EN'
$source = Join-Path $outDir 'aura-trailer-store-1080p-v1.mp4'
$work = Join-Path $repoRoot 'build\trailer\english-subtitles-v2'
if (-not (Test-Path -LiteralPath $source)) { throw 'Original English trailer not found.' }
New-Item -ItemType Directory -Force -Path $work | Out-Null
Copy-Item -LiteralPath 'C:\Windows\Fonts\consola.ttf' -Destination (Join-Path $work 'body.ttf')
$cues = @(
    @{ Start=0; End=3; Text="Your media. Your player.`n[Original instrumental music]" },
    @{ Start=3; End=11; Text='Local audio playback with a live spectrum analyzer.' },
    @{ Start=11; End=19; Text='Play your videos with associated subtitles.' },
    @{ Start=19; End=25; Text='Adjust audio, video, subtitle and interface preferences.' },
    @{ Start=25; End=34; Text='Go fullscreen with floating playback controls.' },
    @{ Start=34; End=37.1; Text='AURA // PLAYER - Audio. Video. Subtitles.' }
)
# A dedicated caption band preserves the entire real app view without cropping.
$filters = @('scale=1760:990:flags=lanczos','pad=1920:1080:80:0:color=0x050e0a','setsar=1','drawbox=x=80:y=994:w=1760:h=1:c=0x46ded7@0.6:t=fill')
for ($i=0; $i -lt $cues.Count; $i++) {
    $cue = $cues[$i]
    [IO.File]::WriteAllText((Join-Path $work "caption-$i.txt"),$cue.Text,(New-Object Text.UTF8Encoding($false)))
    $filters += "drawtext=fontfile=body.ttf:textfile=caption-$i.txt:fontsize=30:fontcolor=white:x=(w-tw)/2:y=1002:line_spacing=6:enable='gte(t,$($cue.Start))*lt(t,$($cue.End))'"
}
$master = Join-Path $outDir 'aura-trailer-store-1080p-en-subs-v2.mp4'
$preview = Join-Path $outDir 'aura-trailer-1080p-en-subs-v2.mp4'
$thumbnail = Join-Path $outDir 'aura-trailer-thumbnail-1920x1080-en-subs-v2.png'
foreach ($output in @($master,$preview,$thumbnail)) {
    if (Test-Path -LiteralPath $output) { throw "Output already exists: $output" }
}
Push-Location $work
try {
    & $ffmpeg -hide_banner -loglevel error -i $source -map 0:v:0 -map 0:a:0 `
        -vf ($filters -join ',') -c:v libx264 -preset medium -profile:v high `
        -pix_fmt yuv420p -r 30 -b:v 50M -minrate 50M -maxrate 50M -bufsize 100M `
        -g 15 -keyint_min 15 -sc_threshold 0 -bf 2 -flags +cgop -coder 1 `
        -x264-params 'nal-hrd=cbr:force-cfr=1:open-gop=0' -tag:v avc1 `
        -c:a copy -movflags +faststart -use_editlist 0 `
        -metadata 'title=Aura Player - English trailer with English subtitles' $master
    if ($LASTEXITCODE -ne 0) { throw 'Subtitled Store trailer render failed.' }
    & $ffmpeg -hide_banner -loglevel error -i $master -c:v libx264 -preset medium -crf 20 -pix_fmt yuv420p -c:a copy -movflags +faststart $preview
    if ($LASTEXITCODE -ne 0) { throw 'Subtitled preview render failed.' }
    & $ffmpeg -hide_banner -loglevel error -ss 1.5 -i $master -frames:v 1 $thumbnail
    if ($LASTEXITCODE -ne 0) { throw 'Subtitled thumbnail render failed.' }
} finally { Pop-Location }
Write-Output "Store: $master"
Write-Output "Preview: $preview"
Write-Output "Thumbnail: $thumbnail"
