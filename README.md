# playerdoit

Reproductor de audio modular en Go. Usa Oto v3 como motor de flujos PCM y
`go-mp3` para decodificar MP3 durante la reproducción.

## Ejecutar

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
