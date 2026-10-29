# Revisión de video, subtítulos y pantalla completa

Fecha: 2026-10-24. Rama: `feature/video-playback`, cambios locales sin commit.
Revisión del código generado en esta sesión; no certificación de todo Aura ni
auditoría de los fuentes C/C++ incorporados en libmpv/FFmpeg.

## Hallazgos

### P2: un subtítulo automático inválido impide abrir un video válido

En `internal/video/player_windows.go:178`, cualquier error de `sub-add` cierra
el reproductor y aborta `Open`. Reproducción confirmada con el MP4 de prueba
válido y un archivo `movie.srt` que contiene texto sin formato SRT:

```
load contextual subtitles: video command sub-add failed: -12
```

Los subtítulos opcionales deberían generar una advertencia y permitir continuar
con el video, cargando las demás pistas válidas.

### P2: operaciones nativas bloqueantes en el hilo de interfaz

Los eventos de la UI llaman sincrónicamente a `Controller.Play` y
`Controller.LoadSubtitles`. `Play` retiene el mutex del controlador y llama a
`video.Open` (`internal/player/controller.go:106`, `:349`). La apertura espera
hasta 15 segundos al archivo y ejecuta comandos nativos síncronos para cada
subtítulo. Esa espera impide procesar eventos gráficos, incluido Esc. El límite
de 15 segundos no cubre todas las llamadas nativas ni el escaneo de directorios.

También faltan límites a la cantidad/tamaño de subtítulos descubiertos. La
consecuencia posible es una ventana sin respuesta ante entradas lentas o
costosas. La ruta de bloqueo se confirma por inspección; no se provocó un cuelgue
prolongado en la aplicación que reproduce la película del usuario.

Recomendación: trabajador de apertura con cancelación y generación por pista,
límites de entradas y subtítulos, y publicación del resultado en la UI.

### P2: búsqueda de dependencias DLL sin endurecer

`internal/video/player_windows.go:82` usa `windows.LoadDLL` con ruta absoluta.
Esto fija la DLL principal, pero no restringe la búsqueda de sus dependencias.
En la versión local de Go, esa función termina en `LoadLibraryExW` con flags 0.
Windows puede buscar dependencias por nombre incluso con una ruta absoluta
para la DLL principal:
https://learn.microsoft.com/en-us/windows/win32/dlls/dynamic-link-library-search-order

Riesgo condicional: una dependencia resoluble desde un directorio controlado por
terceros podría cargar código no deseado. No se ha demostrado una explotación
con la DLL descargada. Usar `LoadLibraryEx` con directorios de búsqueda
restringidos y validar versión/API del runtime. El script verifica SHA-256 del
archivo y de la DLL, pero la carga normal no exige pasar por ese script.

### P2: asociación incorrecta en carpetas de subtítulos compartidas

`internal/video/subtitles.go:32` agrega todos los subtítulos de `Subs` o
`Subtitles`, sin asociarlos al nombre del video. Reproducción confirmada:
`selected-movie.mp4` recibe `Subs/different-movie.spa.srt`. Esa pista incluso puede
seleccionarse por la preferencia de español. La lógica funciona para la carpeta
dedicada a Nobody 2, pero no para una biblioteca con varios videos.

Recomendación: usar coincidencias por video en carpetas compartidas y reservar
la búsqueda general para carpetas inequívocamente dedicadas o selección manual.

## Comprobaciones realizadas

- `go test ./...`: pasó.
- `go vet ./...`: pasó.
- Pruebas reales de video, pausa, búsqueda, volumen, fin, reinicio y cierre: pasaron.
- Prueba con la película real: cuatro archivos cargados como cuatro pistas,
  español seleccionado, diferencias de imagen al ocultar subtítulos y cambio
  a francés verificados.
- Pruebas del router de entrada: F11, Alt+Enter, Esc, restauración de modo,
  doble clic programático, botón de salida, desvanecimiento y movimiento del
  mouse: pasaron. El doble clic programático no sustituye una prueba del gesto
  físico completo.
- `govulncheck` v1.9.0 sobre Windows/amd64 y Go 1.27.1: cero vulnerabilidades
  alcanzables detectadas. Señala además un aviso a nivel de paquetes importados
  y trece a nivel de módulos, sin llamadas vulnerables detectadas. Recomienda
  parches de Go hasta 1.27.3 y `golang.org/x/net` hasta v0.60.0 para los avisos
  respectivos. El escaneo no cubre la DLL nativa.
- `go test -race ./...`: no se pudo ejecutar; CGO está deshabilitado y no se
  encontró compilador C en PATH. La ausencia de carreras no queda acreditada.
- Prueba negativa de red: un archivo local con extensión MP4 y contenido M3U
  apuntando a un servidor HTTP de loopback produjo cero peticiones y fue
  rechazado. Esto solo descarta ese caso concreto.

Las reproducciones de auditoría utilizaron archivos temporales y fueron
retiradas del árbol de fuentes al terminar. No se modificó la implementación
durante esta revisión.

## Aspectos favorables y límites

Los comandos de reproducción no pasan por un shell. El renderizador publica
imágenes inmutables mediante un puntero atómico, y libera el contexto antes de
destruir el motor. Las rutas normales del controlador tienen mutex y el
renderizador no depende de ellos. El script fija versión y hashes del runtime.

La salida usa CPU y una superficie fija de 960×540, también en pantalla completa.
Cada cuadro nuevo reserva aproximadamente 2 MiB: a 60 cuadros/s son unos
119 MiB/s de asignaciones, antes de otras operaciones. No implica por sí mismo
una fuga de memoria; requiere medición de CPU, GC y GPU en sesiones largas.
No se verificó toda la matriz de formatos anunciados ni todas las arquitecturas.
Los decodificadores nativos trabajan dentro del proceso, sin aislamiento.

## Dictamen

Los flujos normales ensayados funcionan, pero el código todavía tiene calidad
de prototipo. No se confirmó una vulnerabilidad crítica explotable; tampoco
hay evidencia suficiente para considerarlo endurecido frente a archivos
hostiles. Corregir primero la tolerancia a subtítulos inválidos y los bloqueos
de UI, restringir la carga de DLL y afinar la asociación de subtítulos antes de
preparar una versión distribuible.


## Correcciones aplicadas - 1.2.1

- Los subtitulos automaticos invalidos se omiten y se siguen cargando los validos. Los fallos de descubrimiento opcional no rechazan el video.
- Apertura, navegacion, carga manual y controles de transporte se despachan fuera del hilo grafico, con una sola operacion simultanea. Durante la operacion se omiten lecturas bloqueantes del controlador; Esc y los eventos de ventana siguen procesandose. No se implementa cancelacion forzada de llamadas nativas ni cambio de pista durante una apertura pendiente.
- LoadLibraryEx restringe dependencias al directorio de la DLL y System32; se exige API cliente libmpv major 2 antes de registrar funciones.
- En bibliotecas compartidas se exige coincidencia con el nombre del video. Los nombres de idioma se aceptan solo en carpetas con un unico video, para conservar el caso Nobody 2.
- Limites: 4096 entradas por directorio, 32 subtitulos automaticos y 16 MiB por subtitulo automatico/manual. Escaneo no recursivo. Estos limites no son un sandbox del decodificador nativo.
- Corregido el cierre duplicado del archivo en el fallback de audio.

Las limitaciones de renderizado, aislamiento nativo y firma digital indicadas arriba siguen vigentes.

Validacion 1.2.1: go test ./..., go vet ./... y Garble test ./... completados; incluye pelicula Nobody 2 y fixtures de audio/video. Pruebas de regresion para subtitulos corruptos, asociacion compartida, limites y despacho asincrono. Esc verificado por router de eventos con una operacion marcada en curso. Instalacion aislada y hashes del payload verificados. No se realizo una nueva inspeccion visual automatizada.
