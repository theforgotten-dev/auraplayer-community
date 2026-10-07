# playerdoit

Reproductor de audio modular en Go. Usa Oto v3 como motor de flujos PCM y
`go-mp3` para decodificar MP3 durante la reproducción.

## Aura Desktop

La rama de UI incluye una aplicación de escritorio construida con Wails v2 y
un frontend HTML/CSS/JavaScript embebido. No requiere Node para compilar.

```powershell
go run .
```

Para generar el ejecutable de producción:

```powershell
go run github.com/wailsapp/wails/v2/cmd/wails@v2.16.0 build -clean
```

El resultado queda en `build/bin/aura-player.exe`. En Windows se requiere el
runtime de WebView2. Desde la aplicación se pueden agregar varios MP3, navegar
la cola, reproducir, pausar, detener, buscar una posición y ajustar el volumen.

Atajos: `Espacio` reproduce o pausa, `j`/`k` recorren la cola, `Enter` reproduce
la selección y `Ctrl+O` abre el selector de archivos.

## CLI

Los archivos MP3 se detectan por su extensión, se decodifican por streaming y
se envían como PCM al motor:

```powershell
go run ./cmd/player ./audio.mp3
```

También se admite PCM crudo. Sus valores predeterminados son 44.1 kHz, estéreo
y signed 16-bit little-endian:

```powershell
go run ./cmd/player ./audio.pcm
```

También se pueden indicar el formato del flujo:

```powershell
go run ./cmd/player -rate 48000 -channels 1 -format f32le ./audio.pcm
```

Formatos disponibles: `s16le`, `u8` y `f32le`.

## Compilar

```powershell
go build -o bin/playerdoit.exe ./cmd/player
```

## Verificar

```powershell
go test ./...
```
