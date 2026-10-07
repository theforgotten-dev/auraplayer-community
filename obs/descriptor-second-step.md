# 🎨 Aura Music Player - Guía de Diseño UI/UX

Este documento define la filosofía de diseño, los principios de experiencia de usuario (UX) y los wireframes conceptuales para la interfaz de **Aura Music Player**, fusionando la eficiencia funcional de los reproductores clásicos con las tendencias visuales modernas.

---

## 🏛️ 1. Filosofía de Diseño: Clásico + Moderno

El diseño de la interfaz recupera las mejores lecciones de los reproductores históricos y las adapta a los estándares visuales actuales (modo oscuro nativo, tipografías limpias y baja fricción de uso).

### Herencia Clásica
* **Modularidad a lo Winamp:** Control absoluto del espacio a través de paneles independientes o acoplables, sin menús ocultos ni capas de navegación profundas. Todo está a un clic o a una tecla de distancia.
* **Contexto Enriquecido a lo Amarok:** Fusión de la reproducción musical con información contextual en tiempo real (metadatos detallados, carátulas y analizadores de espectro alimentados directamente por el motor de audio).
* **Eficiencia de Datos a lo Foobar2000:** Densidad de información optimizada para que el melómano o desarrollador vea exactamente lo que necesita sin distracciones visuales.

### Estándares Modernos
* **Minimalismo Geométrico & Glassmorphism:** Eliminación de texturas hiperrealistas pesadas por bordes finos, transparencias sutiles y una paleta de colores cohesiva (inspirada en temas como *Catppuccin* o *Tokyo Night*).
* **Teclado como Ciudadano de Primera:** Optimización total para navegación mediante atajos de teclado (`vim-bindings` como `j`/`k` para listas, espacio para reproducir/pausar, `/` para búsqueda instantánea).

---

## 📐 2. Wireframe Conceptual: Layout de Tres Paneles

La interfaz se divide en tres zonas lógicas para equilibrar la gestión de la música, el feedback visual del motor y la inspección de metadatos.

```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ AURA ENGINE v1.0 [PCM 44.1kHz / Active Codec: FLAC]                    [—] [□] [✕]     │
├──────────────────────────────┬─────────────────────────────────┬───────────────────────┤
│ PLAYLIST (Queue - 12)        │ NOW PLAYING & VISUALIZER        │ METADATA & SPECS      │
│ 01. Stan Getz - Ipanema      │                                 │ Artist: Miles Davis   │
│ 02. Dave Brubeck - Take Five │   ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░   │ Album: Kind of Blue   │
│>03. Miles Davis - So What    │     ▂▃▄▅▆▇█▇▆▅▄▃▂     ▂▃▄▅      │ Year: 1959            │
│ 04. Coltrane - Giant Steps   │   ( Live PCM Spectrum FFT )     │ Bitrate: 1411 kbps    │
│ 05. Portishead - Glory Box   │                                 │ Format: Lossless      │
├──────────────────────────────┴─────────────────────────────────┴───────────────────────┤
│ [◄◄]  [■ STOP]  [► PLAY / PAUSE]  [►►]   [⇄ SHUF]  [↺ LOOP]   VOL ▬▬▬▬▬▬▬●▬▬  80%     │
└────────────────────────────────────────────────────────────────────────────────────────┘

# 🎨 Aura Music Player (playerdoit) - Guía Definitiva de UI/UX y Arquitectura de Interfaz

Este documento establece la especificación de diseño, experiencia de usuario (UX) y contratos de integración para la interfaz del reproductor, resolviendo la tecnología objetivo, la gestión de paneles, el manejo de errores y la accesibilidad.

---

## 🏛️ 1. Definición Tecnológica y de Marca

* **Nombre del Proyecto:** `playerdoit` (Identificador de repositorio / backend en Go).
* **Nombre Comercial / Marca:** **Aura Music Player**.
* **Tecnología de Interfaz (GUI vs. TUI):** Se adopta **GUI de escritorio** como tecnología principal (utilizando **Wails** con frontend web HTML/CSS/JS o **Fyne**) para cumplir plenamente con la estética de **Glassmorphism**, transparencias sutiles, esquinas redondeadas y ventanas flotantes/acoplables. Se descartó Bubble Tea para la interfaz principal debido a las limitaciones de la terminal con las capas visuales avanzadas.

---

## 🪟 2. Sistema de Paneles Independientes y Acoplables

Inspirado en la modularidad de Winamp y Foobar2000, el espacio de trabajo se compone de 3 paneles lógicos que pueden interactuar de la siguiente forma:

* **Acoplamiento Magnético (*Docking*):** Por defecto se ordenan en una rejilla de tres columnas. Al arrastrar el borde de un panel hacia otro, se ajustan automáticamente (*snapping*).
* **Redimensionamiento:** Cada panel cuenta con cursores de arrastre en sus esquinas y divisores centrales. Su tamaño se almacena en la configuración local del usuario al cerrar la app.
* **Minimizar, Ocultar y Restaurar:**
  * Cada panel incluye controles en su barra de título: `[-]` para colapsar a una barra flotante compacta y `[✕]` para cerrarlo por completo.
  * Los paneles ocultos pueden restaurarse instantáneamente mediante un menú flotante de accesos rápidos (`Ctrl + M` para abrir el gestor de paneles).

---

## 🔌 3. Contrato UI ↔ Backend (Interfaces de Consumo)

Para que la UI se comunique de forma desacoplada con el motor en Go (`playerdoit`), la interfaz consumirá los siguientes contratos de datos:

```go
// Contrato expuesto al motor para que la UI obtenga el estado en tiempo real
type UIController interface {
    GetPlaybackState() PlaybackState         // Playing, Paused, Stopped, Loading, Error
    GetQueue() []Track                       // Lista de reproducción actual y pista activa
    GetMetadata() TrackMetadata              // Artista, Álbum, Año, Artwork (si existe)
    GetVisualizerData() []float32            // Buffer PCM procesado para espectro (Roadmap futuro)
}

type PlaybackState string
const (
    StateStopped   PlaybackState = "STOPPED"
    StatePlaying   PlaybackState = "PLAYING"
    StatePaused    PlaybackState = "PAUSED"
    StateLoading   PlaybackState = "LOADING"
    StateError     PlaybackState = "ERROR"
)