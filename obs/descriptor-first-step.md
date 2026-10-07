Arquitectura Evolutiva de Reproductor de Música en Go
Este documento presenta una propuesta modular y evolutiva para construir un reproductor de música en Go (Golang), diseñado por fases desde el núcleo de audio hasta interfaces de control.
---
🏗️ Fases de Desarrollo (Módulos)
Etapa 1: El Núcleo de Audio (Audio Engine Core)
Abstracción de la reproducción de sonido independiente de la lógica de negocio.
Biblioteca seleccionada: `github.com/ebitengine/oto/v3`.
Componentes clave:
`PlayerEngine`: Interfaz genérica (`Play`, `Pause`, `Stop`, `Seek`, `SetVolume`).
`Context Manager`: Inicialización del driver de audio (frecuencia, canales, bits).
`Buffer Pipeline`: Gestión del flujo de memoria para evitar interrupciones (stuttering).
Etapa 2: Decodificadores y Soporte de Formatos
Adaptadores para traducir archivos comprimidos o estructurados a flujo PCM.
Bibliotecas por formato:
WAV: `github.com/go-audio/wav`
MP3: `github.com/hajimehoshi/go-mp3`
FLAC: `github.com/mewkiz/flac`
OGG / Vorbis: `github.com/jfreymuth/oggvorbis`
Componentes clave:
`Decoder Interface`: Contrato común de lectura y metadatos.
`Format Factory`: Patrón diseño Factory para detección automática de formatos.
Etapa 3: Sistema de Control y Cola (Playlist & State Machine)
Lógica de control de reproducción y gestión de listas.
Componentes clave:
`Playlist Manager`: Cola de reproducción (siguiente, anterior, añadir, remover).
`State Machine`: Control explícito de estados (`Stopped`, `Playing`, `Paused`, `Buffering`).
`Dispatcher`: Sistema de eventos para notificar cambios de estado o tiempo.
Etapa 4: Procesamiento de Señal (DSP / Audio Effects)
Manipulación opcional del flujo de audio en tiempo real.
Componentes clave:
`Volume & Pan Control`: Ganancia y balance estéreo.
`Equalizer`: Filtros de bandas (graves, medios, agudos).
`Visualizer Data Provider`: Extracción de amplitud/frecuencia para visualizadores.
Etapa 5: Interfaces de Usuario (UI / Controllers)
Desacoplamiento de la lógica mediante múltiples capas de control.
Componentes clave:
`CLI Controller`: Comandos por terminal (`github.com/spf13/cobra`).
`TUI`: Interfaz interactiva en consola (`github.com/charmbracelet/bubbletea`).
`REST / RPC API`: Control remoto mediante red.
---
📂 Estructura de Directorios (Go-idiomatic)
```text
my-music-player/
├── cmd/
│   └── player/
│       └── main.go          # Punto de entrada
├── internal/
│   ├── audio/
│   │   ├── engine.go        # Core de Oto/Beep
│   │   └── pipeline.go      # Gestión de buffers
│   ├── codecs/
│   │   ├── decoder.go       # Interfaz común de decodificación
│   │   ├── mp3.go           # Adaptador MP3
│   │   ├── wav.go           # Adaptador WAV
│   │   └── factory.go       # Selector automático
│   ├── player/
│   │   ├── state.go         # Máquina de estados
│   │   └── playlist.go      # Gestión de cola
│   └── ui/
│       └── tui/             # Interfaz de terminal
└── go.mod
