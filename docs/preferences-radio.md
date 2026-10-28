# Preferencias musicales y radios online

Primera versión: Configuración (Ctrl+P) → Descubrir. Escribir un estilo o tema
(jazz, rock, ambient, news), o elegir una sugerencia. «Me interesa +» aumenta
la afinidad; «Menos de esto» la reduce. Pulsa Guardar para conservar los cambios
en el perfil local de Aura (`preferences.json`). Cerrar descarta las ediciones.

En Configuración → Radios puedes elegir Radio Browser o Icecast, editar la URL
de la API o del catálogo XML compatible, cambiar los estilos iniciales separados
por comas, limitar los resultados (10/20/30), configurar un país de dos letras
(solo Radio Browser), ocultar emisoras reportadas como caídas (solo Radio Browser)
y activar o desactivar las sugerencias personalizadas. Las búsquedas permiten
probar la configuración pendiente; Guardar la conserva para el próximo inicio.
«Restablecer radios y perfil» restaura únicamente esta sección y las valoraciones;
«Restablecer» restaura todas las preferencias. Ambos requieren Guardar.

Valores iniciales: Radio Browser, `https://de1.api.radio-browser.info`, todos los
países, 30 resultados, filtro de caídas y personalización activados; estilos
rock, pop, jazz, classical, electronic, ambient, latin y blues. Las preferencias
de versiones anteriores reciben estos valores sin perder sus otros ajustes.

«Buscar radios» consulta el directorio elegido; «Sintonizar» reproduce mediante
libmpv. Requiere el mismo `libmpv-2.dll` de la reproducción multimedia de Windows.
El reproductor cierra la fuente anterior y reutiliza un único espacio de radio,
conservando los archivos de la cola. Detener y volumen usan los controles habituales.
No se permite buscar una posición dentro de una emisión en vivo.

## Decisión: perfil explícito por contenido

Se eligió un modelo pequeño, interpretable y local, inspirado en el filtrado
basado en contenido. Cada estilo tiene un peso entero entre -10 y 10.
Las sugerencias de estilos se ordenan por peso; las estaciones por la media de
los pesos de sus etiquetas normalizadas. Los empates conservan el orden del
directorio (popularidad en Radio Browser; orden del catálogo en Icecast).
Sin valoraciones se muestran los estilos iniciales configurados. Al desactivar
personalización se conserva el orden del directorio y de los estilos editados.
No se infieren gustos a partir de una reproducción accidental.

Esto recomienda estilos y estaciones; todavía no recomienda canciones locales,
no aprende automáticamente de la duración de escucha y no usa modelos remotos.
No requiere cuentas, un servicio de aprendizaje ni nuevas dependencias.
El texto de búsqueda se envía al directorio; las valoraciones permanecen locales.

## Fuentes consultadas

- Modelo por contenido: https://developers.google.com/machine-learning/recommendation/content-based/basics
- Radio Browser: https://docs.radio-browser.info/ — búsqueda con `tag`,
  `hidebroken=true`, máximo 30 resultados y reproducción de `url_resolved`.
- Icecast: https://dir.xiph.org/ — catálogo XML integrado desde
  `https://dir.xiph.org/yp.xml`, con búsqueda local por nombre/estilo, deduplicación
  y descarte de entradas sin URL. No publica país ni estado de comprobación.

Limitaciones: no hay descubrimiento automático de réplicas; la URL se puede
editar. El estado del directorio no garantiza que una emisión esté
disponible: puede cambiar, bloquear regiones o dejar de transmitir. No se extrae
el título de la canción de los metadatos ICY en esta versión. El espectro FFT
actual no analiza el audio reproducido por libmpv.

## Verificación

`go test ./...`, `go vet ./...`, compilación de `build/bin/aura-radio.exe`.
Pruebas del cliente HTTP, cancelación, filtrado/deduplicación, ranking y perfil
persistido. Prueba opcional real en PowerShell:

```powershell
$env:AURA_MPV_DLL = (Resolve-Path build/release-runtime/libmpv-2.dll).Path
$env:AURA_RADIO_LIVE_TEST = '1'
go test ./internal/player -run TestRadioLiveIntegration -v -count=1 -timeout 100s
# Para probar el directorio y la reproduccion con Icecast:
$env:AURA_RADIO_PROVIDER = 'icecast'
go test ./internal/player -run TestRadioLiveIntegration -v -count=1 -timeout 100s
```

El 28 de octubre de 2026 avanzaron Classic Vinyl HD y Anon.FM, con un solo
espacio de radio tras el cambio. La prueba usa volumen cero: confirma reproducción
y avance del motor, no una escucha humana. Las emisoras de la prueba dependen
de los resultados actuales del directorio.

La ampliación configurable también pasó las pruebas de migración de preferencias,
persistencia de los campos editados, aislamiento del borrador y tamaño del panel.
La búsqueda real de Icecast devolvió URLs utilizables y su prueba de reproducción
confirmó avance consecutivo de ICE RADIO WALDKRAIBURG 1 Mix Spezial y Agaune Radio,
manteniendo un único espacio de radio. No se realizó una revisión visual manual
de la interfaz en esta ampliación.
