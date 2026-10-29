# Reconstruccion de Aura tras deteccion en Store

Fecha: 26 de octubre de 2026. Rama: feature/installer-releases.

El aviso aportado por el titular informa deteccion de malware en aura-native.exe
del paquete Aura-Player-Store-1.2.1.0-x64.msix. No incluye nombre de amenaza ni
datos que permitan atribuir la causa a una dependencia, ofuscacion o infeccion.

Por instruccion expresa del titular, se retiro Garble de la compilacion.
Version nueva: EXE 1.2.3, MSIX 1.2.3.0 x64, misma identidad de Partner Center.
Compilacion: go build -trimpath -ldflags "-H=windowsgui". Sin ofuscador, modo
tiny, eliminacion de simbolos o eliminacion explicita del identificador de build.
Los artefactos anteriores se conservaron; no se modificaron exclusiones o
configuraciones de seguridad de Windows.

## Evidencia

- go test ./... y go vet ./... aprobados por el script de construccion.
- Pruebas con fixtures reales de audio y video, usando el runtime LGPL fijado.
- Informacion de Go legible mediante go version -m en el EXE final.
- Instalacion EXE aislada verificada con hashes del payload.
- MSIX validado por MakeAppx y hashes internos comparados con el payload.
- Registro de desarrollo temporal con AppxManifest.xml: estado Ok, version
  1.2.3.0 y PFN 16137272-2.Aura-Player_catmp5kz908t0.
- Arranque mediante shell:AppsFolder: proceso desde el payload correcto,
  ventana Aura Player y Responding=True. Despues se cerro ese proceso y se
  retiro el registro temporal. Esta prueba no es una instalacion firmada de
  Store ni una certificacion WACK.
- Defender informa AntivirusEnabled=False y AMServiceEnabled=False; no se pudo
  repetir un analisis antivirus local ni se cambio ese estado.
- Ningun binario se envio automaticamente a servicios externos de analisis.

SHA-256 del EXE 1.2.3:
8f0190185c42a8a72253f5980c3539d87db97efe1ae76a39de2af8fd48f13902

SHA-256 del MSIX 1.2.3.0:
7f82d9c22a395e03ccd538c8a40254f50b902c5078db94402868a916c6eebe47

Ruta para nuevo envio:
E:\proys\playerdoit\dist\Aura-Player-Store-1.2.3.0-x64.msix

La ausencia de ofuscacion no demuestra que la deteccion anterior fuese falsa
ni garantiza un resultado limpio del nuevo escaneo de Store. Si persiste,
solicitar nombre y detalle de deteccion y usar el proceso oficial de revision:
https://www.microsoft.com/en-us/wdsi/filesubmission
