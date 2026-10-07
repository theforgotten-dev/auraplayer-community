# playerdoit

Reproductor de audio modular escrito en Go. Usa Oto v3 como motor PCM,
`go-mp3` para decodificación por streaming y Gio para una interfaz de escritorio
renderizada directamente por GPU. No utiliza WebView, HTML ni JavaScript.

## Aura Desktop

La aplicación permite agregar varios MP3 mediante el selector nativo del
sistema o arrastrándolos desde Explorer sobre cualquier punto de la ventana.
Los archivos soltados se validan, se agregan a la cola y el primero comienza a
reproducirse cuando la cola estaba vacía. También permite navegar la cola,
reproducir, pausar, detener, buscar una posición y ajustar el volumen.

```powershell
go run .
go run . ".\bin\audiofiles\02 - Legacy.mp3"
```

Cuando se recibe una ruta por línea de comandos, Aura la agrega y comienza a
reproducirla. Esto permite asociaciones de archivo y pruebas directas.

Para crear un ejecutable de Windows sin consola y con el icono propio de Aura:

```powershell
go generate .
go build -ldflags "-H=windowsgui -s -w" -o build/bin/aura-native.exe .
```

`go generate .` reconstruye los recursos `rsrc_windows_*.syso` desde
`assets/aura-icon.png`; solo hace falta ejecutarlo cuando cambie el icono. El
ejecutable generado usa ese recurso en Explorer, el selector de ventanas y la
barra de tareas.

Todas las pistas se normalizan por streaming a PCM estéreo de 48 kHz antes de
entrar a Oto. Una misma cola puede mezclar archivos con frecuencias de muestreo
diferentes.

El monitor central calcula una FFT de 2048 muestras sobre el PCM normalizado y
dibuja 16 bandas logarítmicas entre 40 Hz y 20 kHz. Los canales izquierdo y
derecho se procesan y representan por separado. La interfaz pide nuevos frames
60 veces por segundo mientras el audio se reproduce o decaen las barras.

## Skins nativas

El botón `[ SKINS ]` abre el gestor de apariencias. Incluye `PHOSPHOR_GREEN`,
`AMBER_CRT` y `CYBER_VIOLET`, además de importar manifiestos JSON. La skin activa
se guarda en la carpeta de configuración del usuario.

Un ejemplo completo está disponible en `skins/ice-terminal.json`:

```json
{
  "version": 1,
  "name": "MI_SKIN",
  "author": "Nombre",
  "colors": {
    "background": "#050505",
    "panel": "#08100d",
    "accent": "#00ff88",
    "channelLeft": "#00ff88",
    "channelRight": "#00ddff"
  },
  "metrics": {
    "spacing": 8,
    "cornerRadius": 0,
    "strokeWidth": 1
  }
}
```

Los colores disponibles son `background`, `panel`, `panelRaised`, `line`,
`lineHot`, `text`, `muted`, `accent`, `channelLeft`, `channelRight` y `danger`.
Se aceptan valores `#RRGGBB` y `#RRGGBBAA`. El manifiesto se valida antes de
aplicarse y tiene un límite de 256 KiB.

## CLI

Los MP3 se detectan por extensión, se decodifican por streaming y se envían al
motor como PCM:

```powershell
go run ./cmd/player ./audio.mp3
```

También se admite PCM crudo. Los valores predeterminados son 44.1 kHz, estéreo
y signed 16-bit little-endian:

```powershell
go run ./cmd/player ./audio.pcm
go run ./cmd/player -rate 48000 -channels 1 -format f32le ./audio.pcm
```

Formatos PCM disponibles: `s16le`, `u8` y `f32le`.

## Verificar

```powershell
go test ./...
go vet ./...
```
