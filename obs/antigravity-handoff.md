# Handoff para Antigravity

Fecha: 2026-10-07 (America/Santiago)

## Estado del repositorio

- Rama activa: `feature/gio-ui`
- Commit funcional principal: `c2e6149 Replace WebView UI with native Gio desktop`
- Punto de partida del audio: `b2b3253 Normalize mixed sample rates for playback`
- El prototipo anterior basado en Wails/WebView se conserva en la rama
  `feature/skins-spectrum`, commit `088e116`.
- La rama activa reemplaza completamente Wails por Gio. No requiere WebView,
  HTML, CSS, JavaScript ni Node.

## Objetivo actual

Aura es un reproductor MP3 nativo escrito en Go con:

- decodificación por streaming mediante `go-mp3`;
- salida PCM mediante Oto v3;
- normalización de distintas frecuencias de muestreo a 48 kHz estéreo S16LE;
- FFT estéreo de 2048 muestras;
- 16 bandas logarítmicas entre 40 Hz y 20 kHz;
- interfaz GPU con Gio;
- skins nativas importables desde JSON.

## Arquitectura

- `internal/audio/otoengine`: contexto y reproductores Oto.
- `internal/audio/pipeline`: remuestreo por streaming a 48 kHz.
- `internal/codecs`: selección de decoder y adaptación de `go-mp3`.
- `internal/dsp`: analizador FFT que deja pasar el PCM sin modificarlo.
- `internal/player`: cola, transporte, posición, volumen y coordinación del
  pipeline.
- `internal/ui/native/app.go`: ventana, layout, controles, selector nativo de
  archivos y dibujo del espectro.
- `internal/ui/native/skin.go`: manifiesto, validación, paletas integradas y
  persistencia de skins.
- `skins/ice-terminal.json`: ejemplo de skin externa.

El frontend consulta el espectro directamente en memoria. `dsp.Spectrum` usa
arreglos fijos para evitar slices nuevos en cada frame. Gio sólo solicita 30
FPS durante reproducción; el estado general se actualiza cada 200 ms.

## Ejecutar y compilar

Entorno usado: Windows amd64, Go 1.27.1, `CGO_ENABLED=0`.

```powershell
go run .
go run . ".\bin\audiofiles\02 - Legacy.mp3"
go build -ldflags "-H=windowsgui -s -w" -o build/bin/aura-native.exe .
```

Pasar uno o más MP3 como argumentos agrega la cola y reproduce el primero.
También se pueden cargar varios archivos desde el botón `[ LOAD MP3 ]`, que
usa `gioui.org/x/explorer` y el diálogo nativo de Windows.

## Verificación realizada

```powershell
go test ./...
go vet ./...
```

Ambos comandos pasan en `c2e6149`.

Las pruebas de `internal/dsp` comprueban:

- que el PCM sale byte por byte sin cambios;
- separación de un tono de 1 kHz en el canal izquierdo y 4 kHz en el derecho;
- lecturas que no están alineadas a frames de cuatro bytes;
- limpieza del espectro después de `Seek`.

El ejecutable se probó con `bin/audiofiles/02 - Legacy.mp3`. El proceso se
mantiene responsivo durante reproducción. El consumo observado cambia según la
residencia de recursos GPU; no se observó crecimiento continuo durante las
mediciones breves.

## Corrección visual reciente

El ecualizador había desaparecido porque el separador flexible entre el tiempo
actual y la duración devolvía el ancho y también toda la altura disponible.
Gio reservaba esa altura para la línea de tiempo y dejaba sólo unos píxeles al
FFT. La corrección está en `layoutTimeline`: el separador ahora devuelve todo el
ancho y altura cero.

El área útil del FFT se limita a 180 px dentro del monitor completo y se centra
verticalmente. El monitor conserva cabecera, cuadrícula, barras L/R y medidores
generales. La restauración del layout se verificó visualmente en
`build/gio-equalizer-restored.png`; los archivos de `build/` no forman parte de
Git.

## Skins

El gestor ofrece `PHOSPHOR_GREEN`, `AMBER_CRT` y `CYBER_VIOLET`. Una skin JSON
puede cambiar:

- `background`, `panel`, `panelRaised`;
- `line`, `lineHot`, `text`, `muted`, `accent`, `danger`;
- `channelLeft`, `channelRight`;
- `spacing`, `cornerRadius`, `strokeWidth`.

La skin se valida, tiene un límite de 256 KiB y se persiste en la carpeta de
configuración del usuario bajo `playerdoit/skin.json`.

## Próxima revisión recomendada

1. Abrir el ejecutable y confirmar el FFT durante reproducción en la pantalla;
   `PrintWindow` puede devolver una captura blanca mientras Gio presenta con la
   GPU, por lo que esa API no es una verificación fiable del frame activo.
2. Revisar visualmente el modal `[ SKINS ]` y la importación de
   `skins/ice-terminal.json`.
3. Añadir atajos de teclado sólo después de estabilizar el layout: espacio,
   J/K, Enter y Ctrl+O todavía no están implementados en Gio.
4. Si se necesita medir rendimiento, registrar CPU, memoria privada y memoria
   GPU durante varios minutos; no comparar únicamente el Working Set del
   proceso Wails, porque WebView2 reparte memoria entre procesos auxiliares.
5. Considerar dibujar etiquetas abreviadas de frecuencia directamente dentro
   de `drawSpectrum`. Una fila rígida con las 16 etiquetas llegó a consumir el
   espacio flexible del gráfico y fue retirada.

## Restricciones importantes

- Oto mantiene una configuración fija de 48 kHz durante toda la vida del
  proceso. No crear un contexto Oto por pista.
- El analizador debe permanecer después del remuestreo y antes de Oto para
  representar exactamente el PCM de salida.
- Las llamadas de UI deben permanecer en el loop de Gio. Los selectores de
  archivo son bloqueantes y por eso se ejecutan en goroutines que notifican a
  la ventana mediante `Invalidate`.
- No volver a introducir un puente de telemetría serializado; el acceso directo
  a `Controller.Spectrum()` es parte del objetivo de rendimiento de esta rama.
