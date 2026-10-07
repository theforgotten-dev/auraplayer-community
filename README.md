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
Todas las pistas se normalizan por streaming a PCM estéreo de 48 kHz antes de
entrar a Oto, por lo que una cola puede mezclar archivos con frecuencias de
muestreo diferentes.

El monitor central calcula una FFT de 2048 muestras sobre el PCM normalizado y
muestra 16 bandas logarítmicas entre 40 Hz y 20 kHz. Los canales izquierdo y
derecho se analizan por separado, incluyendo sus niveles generales. La consulta
visual se realiza de forma independiente al estado de la cola.

### Skins y mods visuales

El botón `[ SKINS ]` permite escoger una de las apariencias incluidas o importar
un manifiesto JSON local. La skin activa se conserva entre ejecuciones. Hay un
ejemplo completo en `skins/ice-terminal.json`:

```json
{
  "version": 1,
  "name": "MI_SKIN",
  "variables": {
    "--bg": "#050505",
    "--green": "#00ff88",
    "--channel-left": "#00ff88",
    "--channel-right": "#00ddff"
  },
  "css": ".panel { border-width: 2px; }"
}
```

`variables` admite propiedades CSS personalizadas y `css` permite modificar la
composición visual completa. Los manifiestos se validan antes de aplicarse y su
tamaño máximo es 256 KiB.

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
