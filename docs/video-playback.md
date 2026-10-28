# Reproducción de video y subtítulos

Aura conserva Gio y el motor Oto para audio. Los videos utilizan libmpv para
decodificar y sincronizar imagen, sonido y subtítulos dentro de la misma ventana.
No se lanza un reproductor externo ni se convierte el archivo completo a PCM.

La implementación inicial funciona en Windows. Acepta MP4, MKV, WebM, AVI,
MOV, M4V, MPG, MPEG y WMV; el códec interno también debe ser compatible con
el runtime instalado. Se admiten videos sin audio y audio multicanal.

## Ejecutar en Windows x64

Desde la raíz del repositorio:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\prepare-video-runtime.ps1
go build -ldflags "-H=windowsgui" -o build/video-runtime/aura-native.exe .
.\build\video-runtime\aura-native.exe "C:\Videos\pelicula.mp4"
```

La DLL se busca junto al ejecutable. Para usar otra ubicación o `go run .`:

```powershell
$env:AURA_MPV_DLL = (Resolve-Path .\build\video-runtime\libmpv-2.dll).Path
go run . "C:\Videos\pelicula.mp4"
```

Sin la DLL, Aura sigue reproduciendo audio y muestra un error al abrir video.
El script de desarrollo descarga un runtime fijado por versión y SHA-256 para
pruebas locales. Los binarios no se guardan en Git. El instalador completo 1.1.0
utiliza un runtime diferente: `scripts/prepare-release-runtime.ps1` obtiene la
variante LGPL publicada en el repositorio de mpv, verifica sus hashes y prepara
sus licencias. El instalador incorpora esa DLL y avisos de los módulos Go.
Referencias:
[API de libmpv](https://github.com/mpv-player/mpv/blob/master/include/mpv/client.h),
[build utilizado](https://github.com/shinchiro/mpv-winbuild-cmake/releases/tag/20261021).

## Subtítulos

- Se cargan automáticamente los archivos que coinciden con el nombre del video,
  incluidos sufijos de idioma, por ejemplo `pelicula.es.srt`.
- También se detectan archivos de la carpeta `Subs` o `Subtitles` situada junto
  al video, aunque sus nombres sean diferentes. Se prefiere la pista con sufijo
  `.spa`, `.es` o `.es-419` cuando existe.
- El botón con el nombre de la pista seleccionada permite recorrer los
  subtítulos detectados para ese video.
- `+ SUBS` permite elegir SRT, ASS, SSA o VTT para el video cargado.
- También puede soltarse un archivo de subtítulos sobre la ventana.
- `CC ON/OFF` muestra u oculta los subtítulos sin detener ni buscar el video.
- Los subtítulos incrustados seleccionados por libmpv también se renderizan.

El estado CC indica si se permite mostrar subtítulos; no garantiza que haya una
pista seleccionada ni texto correspondiente al instante actual. La selección
manual se aplica al video cargado y se reinicia al cambiar de archivo.

## Preferencias básicas

`CONFIG` o `Ctrl+P` abre un panel con categorías Audio, Video, Subtítulos e
Interfaz. Permite guardar volumen inicial, silencio, visibilidad e idioma
preferido de subtítulos y ocultación automática de los controles (1 a 5 segundos).
También ofrece acceso a pantalla completa, skins y ventana compacta.

`Guardar` aplica los valores al reproductor y los conserva en
`%APPDATA%\playerdoit\preferences.json`. `Cerrar` o `Esc` descarta los cambios
sin guardar del panel. `Restablecer` prepara los valores iniciales; se aplican
al pulsar `Guardar`. Los accesos a pantalla completa y apariencia actúan de
inmediato. En pantalla completa, el primer `Esc` cierra el panel y el siguiente
sale de pantalla completa.

## Tamaño de ventana y pantalla completa

- Arrastrar los bordes o las esquinas cambia el tamaño de la ventana.
- El botón de maximizar y el doble clic en la barra de título usan los controles
  habituales de Windows.
- `F11`, `Alt+Enter` o doble clic sobre el video alternan pantalla completa.
- `Esc` o el botón `EXIT FULLSCREEN` vuelven al modo anterior de ventana.
- El botón `F11` junto a los controles de subtítulos ofrece la misma acción.

En pantalla completa el video usa todo el área disponible. Se ocultan la cola,
la ficha del archivo y la barra de título. El transporte flota sobre la imagen:
aparece al mover el mouse y se desvanece después de dos segundos de inactividad.
Permanece visible mientras se usa con el mouse o el video está pausado. `Esc`
siempre permite salir, aunque los controles estén ocultos.

## Decisión de implementación

Se usa la API de renderizado por software de libmpv en una goroutine separada
de sus llamadas de control. La superficie está limitada a 960×540 y se publica
como imagen inmutable para Gio. libmpv conserva la proporción original. Esto
permite integrar video sin cambiar el backend gráfico de Gio, a costa de
decodificación y conversión por CPU y menor resolución de presentación. El
archivo original conserva su resolución. La posición se obtiene del reloj de
libmpv, compartido por audio, video y subtítulos.

El cierre espera al renderizador, libera su contexto y destruye el reproductor.
Cambiar entre audio y video cierra primero la pista anterior. Una futura salida
GPU puede reemplazar esta superficie manteniendo el controlador y sus controles.

## Verificación

```powershell
go test ./...
go vet ./...
$env:AURA_MPV_DLL = (Resolve-Path .\build\video-runtime\libmpv-2.dll).Path
$env:AURA_VIDEO_TEST_FILE = "C:\Videos\prueba-de-8-segundos.mp4"
go test -v ./internal/video -count=1
```

El fixture debe ser corto, tener al menos seis segundos e incluir imagen en
movimiento. Las pruebas reales reproducen sonido y verifican cuadros,
avance, pausa, búsqueda, volumen, fin, reinicio, cierre y diferencias de imagen
al mostrar y ocultar subtítulos, tanto manuales como automáticos.
