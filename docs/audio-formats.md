# Formatos de audio de Aura Player 1.2.0

La interfaz de escritorio registra 55 extensiones. La lista exacta se genera
desde el código mediante `go run ./cmd/mediaformats` y se instala en
`FORMATS.json`; el selector, el arrastre y las asociaciones «Abrir con» utilizan
la misma lista. Una extensión aceptada no garantiza que todas sus variantes
internas puedan decodificarse.

| Familia | Extensiones |
| --- | --- |
| MPEG / AAC / libros de audio | mp3, mp1, mp2, mpa, mpga, aac, adts, m4a, m4b, m4r |
| Ogg / Opus / Speex / WebM audio | ogg, oga, opus, spx, weba |
| Sin pérdida | flac, alac, ape, wv, tta, tak |
| PCM y contenedores de estudio | wav, wave, aif, aiff, aifc, caf, au, snd, w64, rf64, bwf |
| Cine / multicanal | ac3, eac3, dts, dtshd, mka |
| DSD | dsf, dff |
| Telefonía | amr, awb, 3ga |
| Windows / RealAudio / Musepack | wma, ra, mpc, mpp, mp+ |
| Históricos y módulos musicales | 8svx, voc, mod, xm, it, s3m, mtm, umx |

MP3, WAV, FLAC y Vorbis conservan los adaptadores PCM y su analizador de
espectro cuando son compatibles. Si el decodificador no admite el archivo, se
intenta el motor multimedia. AAC, M4A, WMA, los formatos ampliados y el audio
multicanal utilizan libmpv incluido en la versión instalable. La reproducción
exclusivamente de audio no crea un renderizador de video. En esta ruta se muestra
la información de reproducción en lugar de una FFT que no recibe muestras.

Se verificaron decodificación, avance, pausa, búsqueda, volumen, parada y cierre
con archivos generados en Opus, AIFF, WavPack, TTA, ALAC, AC-3, E-AC-3, MP2,
CAF, AU, W64, Ogg/Opus, Ogg/FLAC, AAC, WMA y WAV de seis canales. También se
probó una cola mixta de motores. Los demás formatos registrados dependen del
decodificador disponible en libmpv/FFmpeg y no tienen todavía un fixture propio
verificado en este proyecto.

No se promete soporte universal: archivos con DRM, formatos propietarios sin
decodificador y MIDI sin sintetizador/banco de sonidos quedan fuera. El PCM
sin cabecera requiere conocer su formato; continúa disponible mediante el CLI
con parámetros explícitos, no por detección automática en el escritorio.

Pruebas reales:

```powershell
$env:AURA_MPV_DLL = (Resolve-Path build/release-runtime/libmpv-2.dll).Path
$env:AURA_AUDIO_TEST_DIR = (Resolve-Path build/audio-format-tests).Path
go test -v ./internal/video ./internal/player -run TestExtendedAudio -count=1
```
