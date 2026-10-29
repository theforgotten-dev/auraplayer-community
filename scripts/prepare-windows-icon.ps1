[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
$destination = Join-Path $repoRoot 'build\branding'
New-Item -ItemType Directory -Force -Path $destination | Out-Null
Add-Type -AssemblyName System.Drawing
$source = [Drawing.Image]::FromFile((Join-Path $repoRoot 'assets\aura-icon.png'))
$sizes = @(16,24,32,48,64,128,256)
$images = @()
try {
    foreach ($size in $sizes) {
        $bitmap = New-Object Drawing.Bitmap($size,$size)
        $graphics = [Drawing.Graphics]::FromImage($bitmap)
        $buffer = New-Object IO.MemoryStream
        try {
            $graphics.Clear([Drawing.Color]::Transparent)
            $graphics.InterpolationMode = [Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
            $graphics.CompositingQuality = [Drawing.Drawing2D.CompositingQuality]::HighQuality
            $ratio = [Math]::Min($size / $source.Width,$size / $source.Height)
            $width = [int]($source.Width * $ratio); $height = [int]($source.Height * $ratio)
            $graphics.DrawImage($source,[int](($size-$width)/2),[int](($size-$height)/2),$width,$height)
            $bitmap.Save($buffer,[Drawing.Imaging.ImageFormat]::Png)
            $images += ,$buffer.ToArray()
        } finally { $buffer.Dispose(); $graphics.Dispose(); $bitmap.Dispose() }
    }
} finally { $source.Dispose() }
# ICO directory with a PNG image for every Windows icon size; retains alpha.
$output = Join-Path $destination 'aura.ico'
$stream = [IO.File]::Create($output)
$writer = New-Object IO.BinaryWriter($stream)
try {
    $writer.Write([uint16]0); $writer.Write([uint16]1); $writer.Write([uint16]$sizes.Count)
    $offset = 6 + 16 * $sizes.Count
    for ($i=0; $i -lt $sizes.Count; $i++) {
        $dimension = $sizes[$i] % 256
        $writer.Write([byte]$dimension); $writer.Write([byte]$dimension)
        $writer.Write([byte]0); $writer.Write([byte]0)
        $writer.Write([uint16]1); $writer.Write([uint16]32)
        $writer.Write([uint32]$images[$i].Length); $writer.Write([uint32]$offset)
        $offset += $images[$i].Length
    }
    foreach ($bytes in $images) { $writer.Write([byte[]]$bytes) }
} finally { $writer.Dispose(); $stream.Dispose() }
Write-Output "Aura Windows icon: $output"
