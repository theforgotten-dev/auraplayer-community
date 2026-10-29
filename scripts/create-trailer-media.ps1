[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
$ffmpeg = Join-Path $repoRoot 'build\video-runtime\ffmpeg.exe'
if (-not (Test-Path -LiteralPath $ffmpeg)) { throw 'FFmpeg is required in build/video-runtime.' }
$source = Join-Path $repoRoot 'build\trailer\source'
New-Item -ItemType Directory -Force -Path $source | Out-Null
Copy-Item -LiteralPath 'C:\Windows\Fonts\consolab.ttf' -Destination (Join-Path $source 'title.ttf')
Copy-Item -LiteralPath 'C:\Windows\Fonts\consola.ttf' -Destination (Join-Path $source 'body.ttf')
[IO.File]::WriteAllText((Join-Path $source 'motion-title.txt'),'AURA / MOTION STUDY')
[IO.File]::WriteAllText((Join-Path $source 'motion-caption.txt'),'Original demo footage / created for Aura Player')
$audio = '0.10*sin(2*PI*110*t)*(0.65+0.35*sin(2*PI*2*t))+0.055*sin(2*PI*220*t)+0.045*sin(2*PI*(440+110*floor(mod(t*4,4)))*t)*exp(-6*mod(t,0.25))+0.025*sin(2*PI*1760*t)*exp(-12*mod(t,0.5))+0.015*sin(2*PI*3520*t)*exp(-20*mod(t,0.25))'
$video = "nullsrc=s=960x540:r=30:d=90,geq=r='5+12*(sin(X/110+T*0.7)+1)/2':g='13+85*(sin(X/150+Y/190+T*0.45)+1)/2':b='12+65*(sin(X/180-Y/210+T*0.6)+1)/2',drawgrid=w=80:h=60:t=1:c=0x44f5a0@0.12,drawbox=x=40:y=40:w=880:h=460:t=1:c=0x44f5a0@0.5,drawbox=x=350:y=255:w=64:h=125:c=0x44f5a0:t=fill,drawbox=x=448:y=175:w=64:h=205:c=0x46ded7:t=fill,drawbox=x=546:y=285:w=64:h=95:c=0x44f5a0:t=fill,drawbox=x=350:y=405:w=260:h=12:c=0x46ded7:t=fill,drawtext=fontfile=title.ttf:textfile=motion-title.txt:fontsize=38:fontcolor=0x44f5a0:x=(w-tw)/2:y=85,drawtext=fontfile=body.ttf:textfile=motion-caption.txt:fontsize=17:fontcolor=white@0.7:x=(w-tw)/2:y=455,format=yuv420p"
$subs = @'
1
00:00:00,000 --> 00:00:20,000
Your video. Your subtitles.

2
00:00:20,000 --> 00:00:40,000
Stay in control of your playback.

3
00:00:40,000 --> 00:01:10,000
Made for your local media library.

4
00:01:10,000 --> 00:01:30,000
Aura Player / Original demonstration
'@
[IO.File]::WriteAllText((Join-Path $source 'Motion - Original Demo.eng.srt'),$subs)
Push-Location $source
try {
    & $ffmpeg -hide_banner -loglevel error -y -f lavfi -i "aevalsrc='$audio':s=48000:d=90" -af 'pan=stereo|c0=c0|c1=0.85*c0' -c:a flac -metadata 'title=Aurora - Original Demo' 'Aurora - Original Demo.flac'
    if ($LASTEXITCODE -ne 0) { throw 'Original demo audio render failed.' }
    & $ffmpeg -hide_banner -loglevel error -y -f lavfi -i $video -i 'Aurora - Original Demo.flac' -map 0:v -map 1:a -c:v libx264 -preset veryfast -crf 18 -c:a aac -b:a 160k -shortest -movflags +faststart 'Motion - Original Demo.mp4'
    if ($LASTEXITCODE -ne 0) { throw 'Original demo video render failed.' }
} finally { Pop-Location }
Write-Output "Original trailer media: $source"
