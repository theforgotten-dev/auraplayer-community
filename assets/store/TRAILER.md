# Aura Player — tráiler de Store

Dos montajes de 37 segundos, 1920 × 1080, 30 fps, con títulos EN/ES.
Las escenas son grabaciones reales de Aura Player 1.2.2, no interfaces simuladas.
El panel de preferencias conserva el español de la aplicación actual incluso
en el montaje con títulos ingleses. El video demostrativo y sus subtítulos
internos están en inglés en ambos montajes.

Actualización EN v2: `EN/aura-trailer-store-1080p-en-subs-v2.mp4` agrega
subtítulos ingleses integrados en una banda inferior, sin recortar la vista de
la app. Incluye copia ligera `aura-trailer-1080p-en-subs-v2.mp4`, miniatura
`aura-trailer-thumbnail-1920x1080-en-subs-v2.png` y WebVTT
`aura-trailer-captions-v2.vtt`. La interfaz real grabada se conserva, incluido
el panel de preferencias en español. Los títulos promocionales y subtítulos
de esta versión son ingleses. Se mantuvieron todos los originales v1.

## Archivos por idioma

En `EN` y `ES`:

- `aura-trailer-store-1080p-v1.mp4`: máster para entregar a Store, aprox. 232 MB.
- `aura-trailer-1080p-v1.mp4`: copia ligera de revisión, aprox. 6.5 MB.
- `aura-trailer-thumbnail-1920x1080-v1.png`: miniatura PNG tomada del tráiler.
- `aura-trailer-captions-v1.vtt`: subtítulos descriptivos del tráiler, opcionales.

Títulos sugeridos para la ficha:

- EN: `Aura Player — Audio, video and subtitles`
- ES: `Aura Player — Audio, video y subtítulos`

Para subir: elegir el máster `store`, su miniatura y el título del idioma
correspondiente. Los archivos están preparados localmente; no se han subido
ni enviado a certificación. La copia ligera es para revisión, no el máster de
entrega codificado según la guía de Microsoft.

## Contenido

| Tiempo | Escena |
|---|---|
| 00–03 s | Presentación con el logotipo de Aura |
| 03–11 s | Audio local y analizador de espectro real |
| 11–19 s | Video y subtítulos asociados |
| 19–25 s | Preferencias de reproducción |
| 25–34 s | Pantalla completa; controles flotantes que se ocultan |
| 34–37 s | Cierre de marca |

Se creó un tema instrumental sintético y un clip gráfico de demostración para
esta producción. No se usaron películas o grabaciones musicales de terceros.
Las capturas se limitaron a la ventana de Aura y no incluyen los selectores
de archivos, escritorio, navegador u otras aplicaciones. La música es una
pista de montaje procedente del mismo audio original de demostración; no es
una grabación del audio completo del sistema. No hay narración.

## Producción y validación

- `scripts/create-trailer-media.ps1`: genera el material audiovisual original.
- Tomas reales: `build/trailer/takes`, capturadas mediante FFmpeg/gdigrab con el
  identificador de la ventana seleccionada en Computer Use.
- `scripts/render-store-trailer.ps1`: monta las copias ligeras EN/ES.
- `scripts/encode-store-trailer.ps1`: crea los másteres y sus miniaturas.
- Fuentes e intermedios quedan en `build/trailer`; no se distribuyen tipografías.
- Los MP4 generados se excluyen de Git para evitar versionar másteres grandes.

Máster: MP4, AVC1/H.264 High progresivo, YUV 4:2:0, 50 Mbps, GOP cerrado de
15 cuadros, dos B-frames y CABAC. Audio AAC-LC estéreo, 48 kHz y 384 kbps.
Fast Start activado, listas de edición desactivadas. Ambos másteres se
decodificaron completos sin errores. Se revisaron fotogramas de las cuatro
escenas y la miniatura; esto no equivale a certificación de Store.

Referencia: [requisitos oficiales de tráileres](https://learn.microsoft.com/en-us/windows/apps/publish/publish-your-app/screenshots-and-images).
