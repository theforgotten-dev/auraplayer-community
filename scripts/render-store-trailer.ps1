[CmdletBinding()]
param([ValidateSet('EN','ES','Both')][string]$Language = 'Both')
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
$ffmpeg = Join-Path $repoRoot 'build\video-runtime\ffmpeg.exe'
$work = Join-Path $repoRoot 'build\trailer'
$source = Join-Path $work 'source'
$takes = Join-Path $work 'takes'
foreach ($name in @('01-audio','02-video','03-settings','04-fullscreen')) {
    if (-not (Test-Path -LiteralPath (Join-Path $takes "$name.mp4"))) { throw "Missing real app recording: $name" }
}
Copy-Item -LiteralPath 'C:\Windows\Fonts\consolab.ttf' -Destination (Join-Path $work 'title.ttf')
Copy-Item -LiteralPath 'C:\Windows\Fonts\consola.ttf' -Destination (Join-Path $work 'body.ttf')
$languages = if ($Language -eq 'Both') { @('EN','ES') } else { @($Language) }
$copy = @{
    EN = @('AURA // PLAYER','Your media. Your player.','LOCAL AUDIO / LIVE SPECTRUM','VIDEO / SUBTITLES','PLAYBACK SETTINGS','FULLSCREEN / FLOATING CONTROLS','Audio. Video. Subtitles.','ACTUAL APP CAPTURE')
    ES = @('AURA // PLAYER','Tus archivos. Tu reproductor.','AUDIO LOCAL / ESPECTRO EN VIVO','VIDEO / SUBTITULOS','PREFERENCIAS DE REPRODUCCION','PANTALLA COMPLETA / CONTROLES FLOTANTES','Audio. Video. Subtitulos.','CAPTURA REAL DE LA APP')
}
Push-Location $work
try {
    foreach ($locale in $languages) {
        $texts = $copy[$locale]
        for ($i=0; $i -lt $texts.Count; $i++) { [IO.File]::WriteAllText((Join-Path $work "$locale-$i.txt"),$texts[$i],(New-Object Text.UTF8Encoding($false))) }
        $chains = @()
        $chains += '[1:v]scale=300:300,format=rgba,split[logoIntro][logoEnd]'
        $chains += "[0:v][logoIntro]overlay=(W-w)/2:140:shortest=1,drawgrid=w=120:h=120:t=1:c=0x44f5a0@0.07,drawtext=fontfile=title.ttf:textfile=$locale-0.txt:fontsize=86:fontcolor=0x44f5a0:x=(w-tw)/2:y=515,drawtext=fontfile=body.ttf:textfile=$locale-1.txt:fontsize=36:fontcolor=0x46ded7:x=(w-tw)/2:y=640,drawbox=x=420:y=745:w=1080:h=2:c=0x44f5a0@0.7:t=fill,setsar=1,format=yuv420p,fade=t=in:d=0.3,fade=t=out:st=2.7:d=0.3[intro]"
        $durations = @(8,8,6,9)
        $starts = @(2,1,0.5,0)
        for ($i=0; $i -lt 4; $i++) {
            $inputIndex = $i+2; $textIndex = $i+2; $duration = $durations[$i]; $start = $starts[$i]; $fadeOut = $duration-0.2
            $chains += "[$inputIndex`:v]trim=start=$start`:duration=$duration,setpts=PTS-STARTPTS,fps=30,scale=1824:980:flags=lanczos,pad=1920:1080:48:76:color=0x050e0a,setsar=1,drawbox=x=47:y=75:w=1826:h=982:c=0x44f5a0@0.35:t=1,drawtext=fontfile=title.ttf:textfile=$locale-$textIndex.txt:fontsize=27:fontcolor=0x44f5a0:x=48:y=25,drawtext=fontfile=body.ttf:textfile=$locale-7.txt:fontsize=20:fontcolor=0x46ded7:x=w-tw-48:y=31,format=yuv420p,fade=t=in:d=0.2,fade=t=out:st=$fadeOut`:d=0.2[scene$i]"
        }
        $chains += "[6:v][logoEnd]overlay=(W-w)/2:140:shortest=1,drawgrid=w=120:h=120:t=1:c=0x44f5a0@0.07,drawtext=fontfile=title.ttf:textfile=$locale-0.txt:fontsize=86:fontcolor=0x44f5a0:x=(w-tw)/2:y=515,drawtext=fontfile=body.ttf:textfile=$locale-6.txt:fontsize=36:fontcolor=0x46ded7:x=(w-tw)/2:y=640,drawbox=x=420:y=745:w=1080:h=2:c=0x44f5a0@0.7:t=fill,setsar=1,format=yuv420p,fade=t=in:d=0.25,fade=t=out:st=2.4:d=0.6[ending]"
        $chains += '[intro][scene0][scene1][scene2][scene3][ending]concat=n=6:v=1:a=0[video]'
        $chains += '[7:a]atrim=duration=37,asetpts=PTS-STARTPTS,loudnorm=I=-18:TP=-2:LRA=7,afade=t=in:d=0.5,afade=t=out:st=35.5:d=1.5,aformat=sample_rates=48000:channel_layouts=stereo[audio]'
        $filter = Join-Path $work "edit-$locale.ffscript"
        [IO.File]::WriteAllText($filter,($chains -join ";`n"),(New-Object Text.UTF8Encoding($false)))
        $outDir = Join-Path $repoRoot "assets\store\$locale"
        New-Item -ItemType Directory -Force -Path $outDir | Out-Null
        $output = Join-Path $outDir 'aura-trailer-1080p-v1.mp4'
        & $ffmpeg -hide_banner -loglevel error -y `
            -f lavfi -i 'color=c=0x050e0a:s=1920x1080:r=30:d=3' `
            -loop 1 -framerate 30 -i (Join-Path $repoRoot 'assets\aura-icon.png') `
            -i (Join-Path $takes '01-audio.mp4') -i (Join-Path $takes '02-video.mp4') `
            -i (Join-Path $takes '03-settings.mp4') -i (Join-Path $takes '04-fullscreen.mp4') `
            -f lavfi -i 'color=c=0x050e0a:s=1920x1080:r=30:d=3' `
            -i (Join-Path $source 'Aurora - Original Demo.flac') `
            -filter_complex ($chains -join ';') -map '[video]' -map '[audio]' `
            -t 37 -c:v libx264 -preset medium -crf 18 -pix_fmt yuv420p -r 30 `
            -c:a aac -b:a 192k -ar 48000 -ac 2 -movflags +faststart `
            -metadata 'title=Aura Player - Real app trailer' -metadata "language=$locale" $output
        if ($LASTEXITCODE -ne 0) { throw "Trailer render failed: $locale" }
        Write-Output "Trailer: $output"
    }
} finally { Pop-Location }
