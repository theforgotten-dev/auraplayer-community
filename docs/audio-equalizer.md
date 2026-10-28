# Ecualizador grafico y filtros derivados

Hito superado y aceptado por el usuario el 28 de octubre de 2026.

Acceso: Ctrl+P -> Audio -> Ecualizador y filtros. Diez reguladores verticales
visibles simultaneamente: 31, 63, 125, 250, 500 Hz, 1, 2, 4, 8 y 16 kHz.
Cada banda permite -12 a +12 dB, con 0 dB central. PRE es un regulador vertical
independiente de -24 a +12 dB. Se retiraron las bandas horizontales y los controles
manuales de frecuencia de corte del demo anterior.

Los cambios tienen preescucha en vivo, con actualizaciones agrupadas cada 80 ms.
Guardar conserva el perfil; Cerrar o Escape restaura el sonido guardado. El bloque
esta desactivado por defecto (bypass). Plano, Voz y Graves son puntos de partida
editables. La grafica representa la respuesta calculada, no un espectro medido.

## Cortes automaticos

Regla de producto, no analisis de la musica ni correccion de sala:

- El grupo inferior usa 31/63/125 Hz con pesos 0.6/0.3/0.1.
- El superior usa 16/8/4 kHz con los mismos pesos.
- Para cada grupo, s = clamp((-ganancia_ponderada - 3) / 9, 0, 1).
- Pasa-altos: 125*s*s Hz, redondeado a 0.1 Hz; bajo 1 Hz se desactiva.
- Pasa-bajos: 24000 - 19000*s Hz cuando s > 0; desde 23999 Hz se desactiva.
- Los ajustes planos o con refuerzo en extremos no agregan esos cortes.

La interfaz muestra los cortes derivados. Todos los biquads de EQ y los cortes
comparten estos parametros tanto en la ruta PCM como en libmpv. La reserva de
nivel usa el maximo calculado de la respuesta conjunta en 513 puntos logaritmicos
mas 0.5 dB de margen. Se muestra el preamplificador efectivo. No es un limitador
true-peak; un preamplificador positivo aun puede saturar la salida.

## Implementacion y compatibilidad

Ruta PCM: decodificador -> estereo S16 a 48 kHz -> tono -> FFT -> Oto.
Bandas peaking Q=sqrt(2), cortes Butterworth Q=1/sqrt(2), formulas RBJ.
Estados independientes por canal, almacenamiento fijo, reinicio al buscar y
fundido de 20 ms entre configuraciones. Los coeficientes de actualizacion se
calculan antes de tomar el bloqueo del lector PCM.

Radios y formatos libmpv: aresample/volume/equalizer/highpass/lowpass en lavfi a
48 kHz. El motor reconstruye filtros; no se garantiza el mismo fundido de la
ruta PCM. El FFT de Aura sigue siendo exclusivo de la ruta PCM.

Los perfiles de cinco bandas se migran interpolando sobre frecuencias
logaritmicas; los antiguos cortes manuales se aproximan como atenuaciones en
las diez bandas. La aproximacion puede diferir del demo. Los perfiles nuevos
persisten diez ganancias y no exponen cortes manuales.

## Evidencia y limites

- go test ./... y go vet ./... aprobados.
- Respuesta medida con senos frente a transferencia calculada, aislamiento
  estereo, bypass/plano byte-exacto, seek, polos estables y migracion de perfiles.
- Preescucha/cancelacion/guardado y disposicion del panel cubiertos por pruebas.
- PCM/Oto y libmpv reales: avance, filtros activos, cambio y bypass con volumen cero.
- Revision visual: diez faders y PRE, curva y cortes del preset Voz comprobados.
- Benchmark orientativo: ~3.22 ms por segundo de audio estereo a 48 kHz con diez
  bandas y cortes, cero asignaciones en lectura. No mide CPU total ni glitches.
- No se acredita escucha humana ni prueba con detector de carreras.
- El instalador 1.3.1 pertenece al hito anterior; no incluye este ecualizador.

```powershell
go test ./...
go vet ./...
$env:AURA_MPV_DLL = (Resolve-Path build/release-runtime/libmpv-2.dll).Path
$env:AURA_TONE_INTEGRATION = '1'
go test ./internal/video ./internal/player -run 'TestTone.*Integration' -v -count=1
go test ./internal/dsp -run '^$' -bench BenchmarkToneRead -benchmem
```

Skill personal creada y validada: audio-engineer, en el directorio de skills
del usuario. No es una dependencia de ejecucion ni se instala con Aura.

Fuentes: https://www.w3.org/TR/audio-eq-cookbook/
https://ffmpeg.org/ffmpeg-filters.html
https://mpv.io/manual/stable/#audio-filters