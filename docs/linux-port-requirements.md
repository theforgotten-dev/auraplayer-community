# Aura Player para Linux: requisitos de portabilidad

Este documento es el punto de partida para retomar con Codex el port de Aura
Player desde Windows a Linux. Define el alcance inicial, el sistema de paquetes,
la arquitectura de la aplicación y las pruebas necesarias. No implica que el
port ya esté implementado.

## Objetivo de la primera versión

Publicar una versión Linux de Aura Player que conserve la experiencia principal
de escritorio de Windows:

- Interfaz Gio nativa, reproducción Oto, cola, controles, búsqueda, volumen y
  visualizador estéreo.
- Selector de archivos, arrastrar y soltar y skins.
- Los formatos ya anunciados por la aplicación: MP3, WAV, FLAC, Ogg Vorbis,
  AAC, M4A y WMA. Si un formato no puede ofrecerse en la primera entrega por
  limitaciones verificadas de codecs o licencias, documentarlo y ajustar el
  alcance explícitamente; no anunciar soporte que solo devuelve un error.
- Instalación y desinstalación integradas con el sistema de paquetes Debian.

Mantener la separación actual entre interfaz, controlador, decodificadores,
pipeline PCM, motor de audio y DSP. La capa de plataforma debe resolver
integraciones nativas, no duplicar la lógica del reproductor.

## Plataforma inicial y arquitectura

- **Distribución de referencia:** Ubuntu 24.04 LTS de 64 bits (Noble).
- **Paquete de entrega:** `.deb`, instalable con `apt`/`dpkg`.
- **Arquitectura inicial:** `amd64` / `x86_64`, enlazada contra glibc.
- **Entorno de desarrollo e integración:** WSL2, preferiblemente con Ubuntu
  24.04. No usar compilación cruzada desde Windows como sustituto de una prueba
  nativa.
- **Validación de escritorio:** WSLg puede servir para iterar en UI, audio y
  gráficos, pero no sustituye la verificación final en una sesión Ubuntu Linux
  real con Wayland y X11. Registrar cualquier limitación propia de WSLg.

Debian 12 puede considerarse después como objetivo compatible si sus
dependencias, versiones de glibc, codecs y pruebas se validan por separado. No
prometer una versión mínima de Debian solo porque el `.deb` se pueda construir.
ARM64, otras familias de distribución, Flatpak y AppImage quedan fuera de esta
primera entrega.

## Entorno de desarrollo en WSL2

1. Instalar o seleccionar una distribución Ubuntu 24.04 en WSL2 y confirmar:
   `wsl.exe -l -v`, `uname -m`, `cat /etc/os-release` y `go version`.
2. Usar un SDK de Go que satisfaga el mínimo de `go.mod` (`go 1.27.1`); el
   paquete de Go de la distribución solo sirve si cumple esa versión.
3. Guardar el checkout dentro del filesystem Linux de WSL (por ejemplo,
   `~/src/playerdoit`) para evitar el coste y las diferencias de permisos del
   montaje `/mnt/c`.
4. Instalar las herramientas y cabeceras de desarrollo indicadas abajo.
5. Ejecutar primero `go test ./...` y `go vet ./...`; después compilar y probar
   el ejecutable desde una sesión gráfica y de audio disponible.

Paquetes iniciales para compilar Gio y trabajar con las dependencias nativas en
Ubuntu 24.04:

```bash
sudo apt update
sudo apt install -y \
  build-essential pkg-config git ca-certificates \
  libx11-dev libx11-xcb-dev libxkbcommon-dev libxkbcommon-x11-dev \
  libwayland-dev libegl1-mesa-dev libgles2-mesa-dev \
  libxcursor-dev libxfixes-dev libxrandr-dev libxinerama-dev \
  libxi-dev libxext-dev libxrender-dev
```

Confirmar y ajustar esta lista ejecutando un build limpio en Ubuntu 24.04; no
mantener dependencias de desarrollo que no use la versión final. Los nombres y
las versiones de bibliotecas de audio cambian entre releases. En particular,
Ubuntu 24.04 usa el paquete ALSA `libasound2t64`; en Debian 12 normalmente es
`libasound2`. Evitar copiar nombres de paquete entre distribuciones sin
verificarlos con `apt-cache`.

## Dependencias de ejecución y escritorio

Oto v3 para Linux intenta PulseAudio y puede recurrir a ALSA. El proceso carga
los backends en tiempo de ejecución, por lo que hay que verificar las
dependencias de bibliotecas compartidas y los dispositivos/servidores
disponibles, no solo que la compilación termine.

- Probar la salida PulseAudio, incluido PipeWire con compatibilidad PulseAudio,
  y el fallback ALSA.
- No instalar ni iniciar un servidor de audio dentro del paquete. La sesión del
  usuario proporciona su servicio de audio.
- Gio proporciona ventanas Wayland/X11 y renderizado EGL/GLES. Identificar las
  bibliotecas compartidas requeridas por el binario final y declararlas en
  `Depends`; no asumir que los paquetes `-dev` deben instalarse al usuario.
- El selector de archivos de Gio en Linux usa el portal de escritorio por
  D-Bus. Recomendar el servicio `xdg-desktop-portal` y un backend compatible con
  el escritorio (por ejemplo, GNOME o KDE); probar el comportamiento cuando el
  portal no esté disponible.
- Mantener el arranque y la reproducción de archivos pasados como argumento,
  incluso cuando el selector de archivos no esté instalado.

La lista definitiva de dependencias de ejecución debe salir de una instalación
limpia de Ubuntu 24.04, inspección del binario (por ejemplo `ldd`), ejecución de
pruebas y validación del paquete; no copiar la lista de build dependencies.

## Portabilidad que falta en el código

Antes de añadir código, revisar y reutilizar las interfaces actuales. En
particular:

1. **Decodificación AAC/M4A/WMA:** `internal/codecs/platform_windows.go` usa
   Media Foundation; `internal/codecs/platform_other.go` devuelve un error
   para todo sistema que no sea Windows. Diseñar una implementación Linux
   explícita (build tag `linux`) y ajustar el stub genérico para que no compita
   con ella. Evaluar FFmpeg como proceso externo para decodificar a PCM y
   conservar búsqueda/streaming o cache temporal sin cargar canciones completas
   en memoria. Verificar disponibilidad y nombres de codecs en Ubuntu 24.04 y
   revisar obligaciones de licencias/distribución antes de incluir o exigir
   FFmpeg. No depender accidentalmente de Media Foundation ni afirmar que el
   backend Windows funciona en Linux.
2. **Drag and drop:** `internal/ui/native/file_drop_windows.go` integra el
   mecanismo de Windows y `file_drop_other.go` es un stub. Implementar el
   mecanismo de Gio/Linux, preferentemente compatible con Wayland y X11; probar
   filtros de extensiones, varias rutas, rutas con espacios y errores.
3. **Icono e integración del escritorio:** `window_icon_other.go` es un stub
   apropiado para APIs específicas del taskbar, pero Linux debe empaquetar los
   recursos de icono y un archivo `.desktop` para el menú de aplicaciones.
   Implementar icono de ventana Linux solo si Gio lo requiere; no trasladar
   APIs Win32. Añadir el tipo MIME/asociaciones solo si se implementan con
   metadatos XDG y se documenta el flujo de consentimiento del usuario.
4. **Recursos Windows:** revisar los `.syso`, `go:generate` y los scripts de
   build para asegurarse de que la compilación Linux no los requiera ni
   contamine el paquete Linux.
5. **Configuración y temporales:** usar las convenciones XDG para configuración,
   datos y caché donde corresponda. Comprobar permisos, limpieza de caché PCM y
   comportamiento en rutas Unicode y con espacios.
6. **Selectores:** Gio Explorer tiene implementación Linux basada en portal
   D-Bus; verificar selección múltiple, filtros de formatos, cancelación y
   cierre de recursos en GNOME y KDE.

Los tests generales no deben cambiar el contrato de Windows. Añadir pruebas
específicas de Linux donde el comportamiento de plataforma pueda probarse sin
un escritorio; mantener las pruebas de integración de audio/UI separadas y
documentar requisitos de entorno.

## Empaquetado `.deb` y `apt`

Crear primero un `.deb` local para Ubuntu 24.04 `amd64`; un repositorio APT
firmado y alojado es una etapa posterior. El paquete debe:

- Instalar el ejecutable con un nombre estable, por ejemplo
  `/usr/bin/aura-player`.
- Instalar iconos bajo `/usr/share/icons/hicolor/` y
  `/usr/share/applications/aura-player.desktop`.
- Incluir licencia y documentación bajo `/usr/share/doc/aura-player/`.
- Declarar dependencias de **runtime** en los metadatos `Depends`.
- No ejecutar `sudo`, servicios, descargas ni scripts privilegiados durante la
  compilación. Evitar mantenedores `postinst`/`prerm` salvo necesidad justificada.
- No escribir la configuración del usuario en `/usr`; preservar la skin y
  preferencias al desinstalar.
- Tener nombre, versión, arquitectura, descripción, icono y categoría de
  escritorio consistentes con Aura Player.

Usar inicialmente `dpkg-deb` o una herramienta de empaquetado reproducible como
`nfpm`; fijar la herramienta/versión en el flujo de build y documentar cómo
reconstruir el paquete. Probar el `.deb` en una VM/contendor limpio Ubuntu 24.04:

```bash
sudo apt install ./aura-player_<version>_amd64.deb
dpkg-deb --info ./aura-player_<version>_amd64.deb
dpkg-deb --contents ./aura-player_<version>_amd64.deb
```

También comprobar el menú de aplicaciones, lanzamiento mediante URI/ruta,
reproducción real, cierre y desinstalación (`sudo apt remove aura-player`).
Verificar que datos personales y skins siguen intactos. Para actualizaciones
por `apt install aura-player`, configurar más adelante un repositorio APT
firmado; el `.deb` descargable inicial no requiere ese repositorio.

## Criterios de aceptación del port inicial

- Compila desde un checkout limpio dentro de Ubuntu 24.04 WSL2 `amd64` con el
  SDK Go exigido, y pasa `go test ./...` y `go vet ./...`.
- Se inicia en sesión gráfica; documenta y muestra errores útiles cuando no hay
  audio o portal disponible.
- Se prueba al menos una vez en GNOME/Wayland y X11 fuera de WSLg, o se registra
  explícitamente qué validación queda pendiente.
- El selector, la cola, los controles, la búsqueda, skins y el drag-and-drop
  funcionan sin regresiones en Windows.
- Cada formato listado se prueba con fixture legal y pequeño, incluido el
  comportamiento de búsqueda y el manejo de archivos inválidos.
- El `.deb` declara dependencias correctas, instala menú e icono y se instala,
  ejecuta y desinstala en una imagen limpia Ubuntu 24.04 `amd64`.
- La documentación distingue codecs disponibles de los que dependen de
  paquetes externos y no promete compatibilidad no probada.

## Secuencia sugerida para retomar con Codex

1. Confirmar rama `linux`, estado Git limpio y base Ubuntu 24.04 WSL2 `amd64`.
2. Compilar/tests Linux para sacar una lista real de bloqueos antes de editar.
3. Portar audio y resolver estrategia/dependencias para AAC, M4A y WMA.
4. Implementar y probar drag-and-drop Linux y metadatos XDG de escritorio.
5. Crear el build reproducible y `.deb`, comprobar dependencias en instalación
   limpia y actualizar README.
6. Ejecutar la matriz de aceptación, registrar limitaciones restantes y dejar
   el resultado en commits pequeños y revisables.
