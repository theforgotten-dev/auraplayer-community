# playerdoit

## Licencia

Software privativo. Copyright (c) 2026 Juan Pablo Muñoz Leiva (empowdered).
Todos los derechos reservados. El uso, la copia, la modificación y la
distribución requieren autorización previa y por escrito. Consulta [LICENSE](LICENSE).
Las dependencias y los materiales de terceros conservan sus propias licencias.

Reproductor de audio modular escrito en Go. Usa Oto v3 como motor PCM,
`go-mp3` para decodificación por streaming y Gio para una interfaz de escritorio
renderizada directamente por GPU. No utiliza WebView, HTML ni JavaScript.

## Aura Desktop

La aplicación permite agregar archivos MP3, WAV, FLAC, OGG Vorbis, AAC, M4A y
WMA mediante el selector nativo del sistema o arrastrándolos desde Explorer
sobre cualquier punto de la ventana.
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

Para compilar el instalador de Windows x64 se requiere Inno Setup 6:

```powershell
powershell -ExecutionPolicy Bypass -File .\installer\build-installer.ps1 -Version 1.0.0
```

El instalador queda en `dist/Aura-Music-Player-Setup-1.0.0.exe`, instala Aura
solo para el usuario actual y ofrece crear accesos directos y asociar los
formatos de audio compatibles. La configuración personal de skins no se
elimina al desinstalar.

Todas las pistas se normalizan por streaming a PCM estéreo de 48 kHz antes de
entrar a Oto. Una misma cola puede mezclar formatos, archivos mono o estéreo y
frecuencias de muestreo diferentes.

WAV, FLAC y OGG Vorbis se decodifican mediante los adaptadores nativos de Beep.
En Windows, AAC, M4A y WMA se decodifican mediante Media Foundation. Estos tres
formatos mantienen un cache PCM temporal en disco para permitir búsquedas sin
retener la canción decodificada completa en memoria; el cache se elimina al
cambiar de pista o cerrar el reproductor.

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

Los formatos comprimidos se detectan por extensión, se decodifican por
streaming y se envían al motor como PCM:

```powershell
go run ./cmd/player ./audio.mp3
go run ./cmd/player ./audio.flac
go run ./cmd/player ./audio.m4a
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

## Linux (Debian 13)

El paquete de entrega para Linux es un `.deb` nativo para `amd64`. Los scripts
de empaquetado viven en `packaging/` y generan el binario `/usr/bin/aura-player`,
el lanzador `/usr/share/applications/aura-player.desktop`, los iconos en
`/usr/share/icons/hicolor/` y la documentación en `/usr/share/doc/aura-player/`.

En un sistema Debian 13 (trixie) limpio, instala primero las dependencias de
compilación y luego construye el paquete:

```bash
sudo ./packaging/install-build-deps.sh
./packaging/build-deb.sh 1.0.0 1
```

El resultado queda en `dist/aura-player_1.0.0-1_amd64.deb`. El campo `Depends`
se calcula automáticamente con `dpkg-shlibdeps`; solo la dependencia ALSA
`libasound2t64` se declara a mano porque el motor la carga en tiempo de
ejecución mediante `dlopen`. El paquete no ejecuta `sudo`, servicios ni
`postinst`/`prerm`, y la skin del usuario en `$XDG_CONFIG_HOME/playerdoit` se
conserva al desinstalar.
