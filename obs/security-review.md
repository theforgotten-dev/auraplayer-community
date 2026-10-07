# Revisión rápida de seguridad

Fecha: 2026-10-07

## Alcance

Revisión estática del código Go, entradas de archivos, persistencia local,
gestión de recursos y dependencias alcanzables. Se ejecutaron:

```powershell
go test ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
go run github.com/securego/gosec/v2/cmd/gosec@latest -quiet ./...
```

## Hallazgos corregidos

1. `golang.org/x/image v0.41.0` exponía dos rutas alcanzables reportadas por la
   base oficial de Go: panic con un TIFF malformado y consumo de memoria sin
   límite por tamaños de tiles. Se actualizó `x/image` a `v0.46.0`, `x/text` a
   `v0.42.0` y `x/sys` a `v0.48.0` como dependencia transitiva.
2. El límite de 256 KiB para skins podía eludirse con un primer objeto JSON
   válido seguido de contenido adicional. El parser ahora lee como máximo
   256 KiB, rechaza el byte excedente y exige exactamente un objeto JSON.
3. Los nombres y autores de skins ahora rechazan caracteres de control para
   evitar alteraciones de layout o mensajes engañosos.
4. El directorio de configuración se crea y ajusta a permisos `0700`; el
   archivo de skin continúa escribiéndose con `0600` en plataformas POSIX.
5. Las rutas de error al construir el pipeline ya no descartan fallos al cerrar
   el reproductor o el archivo.
6. Las conversiones entre palabras PCM sin signo y muestras S16LE están
   documentadas como reinterpretaciones intencionales de complemento a dos.

## Superficie observada

- La aplicación no abre puertos, no realiza solicitudes HTTP y no ejecuta
  comandos externos.
- No almacena credenciales ni otros secretos.
- Las rutas de audio provienen del selector del usuario o de argumentos de la
  línea de comandos; abrir rutas arbitrarias es una función explícita.
- Las skins aceptan únicamente campos conocidos, colores hexadecimales y
  métricas dentro de rangos acotados.

## Riesgos residuales

- La decodificación MP3 ocurre dentro del proceso mediante `go-mp3`. No hay una
  vulnerabilidad conocida en `v0.3.4` según `govulncheck`, pero un defecto aún
  desconocido en el parser podría provocar un cierre o consumo excesivo al
  abrir un MP3 manipulado. Aislar el decoder en un proceso con límites sería el
  endurecimiento siguiente si se aceptarán archivos de fuentes hostiles.
- Los ejecutables locales no están firmados. Una distribución pública debería
  incorporar firma de código, hashes reproducibles y un canal de actualización
  autenticado.

## Resultado

Después de las correcciones, `go test`, `go vet`, `govulncheck` y `gosec`
finalizan sin hallazgos.
