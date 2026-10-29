# Instaladores y futuras versiones

Rama de trabajo: `feature/installer-releases`, creada desde
`feature/video-playback` en `ffecc22`. Los cambios locales existentes se
conservaron al cambiar de rama; crear la rama no crea un commit de esos cambios.
No se ha hecho push ni se ha activado un pipeline.

## Generación manual

Desde PowerShell, en la raíz del repositorio, para una nueva versión elegida
explícitamente (ejemplo 1.2.2, todavía no publicada):

```powershell
.\installer\build-installer.ps1 -Version 1.2.2
.\installer\verify-installer.ps1 -Version 1.2.2
```

El primer script prepara el runtime fijado por hash, ejecuta pruebas y vet,
genera recursos Windows, compila con Garble, incorpora textos legales y
licencias, y produce instalador, portable y SHA-256 en `dist`. El segundo
comprueba una instalación aislada con identidad de prueba. Usa una aceptación
técnica del EULA exclusivamente para la prueba automatizada.

Desde 1.2.3, la compilación usa Go estándar sin ofuscación. Se retiraron Garble,
su modo tiny y la generación de semillas de los scripts de entrega, por
instrucción del titular tras el rechazo del escáner de Store. No se requiere
`-PlainBuild`: la compilación estándar es la única ruta actual.

El icono del instalador se genera con `scripts/prepare-windows-icon.ps1` desde
`assets/aura-icon.png`, en tamaños 16 a 256, y se escribe en
`build/branding/aura.ico`. No se usa el antiguo `build/windows/icon.ico`, que
contenía la W heredada. Los iconos MSIX se generan directamente desde el mismo
PNG por `installer/build-msix.ps1`; el MSIX 1.2.1.0 ya usa el logotipo de Aura.

Los scripts actuales permiten sobrescribir artefactos de una misma versión.
Antes de una distribución estable hay que elegir una versión nueva y conservar
el código exacto, las herramientas y los hashes. El instalador 1.2.1 existente
no incorpora el EULA nuevo. No reemplazar un binario ya publicado bajo la misma
URL de versión.

## Pipeline posterior

Diseño propuesto, pendiente de implementar cuando se decida automatizar:

1. Partir de un commit limpio y una etiqueta de versión consistente.
2. Fijar Go, Garble, Inno Setup y runtime; ejecutar validaciones y pruebas reales
   de audio/video en un entorno Windows que permita esos ensayos. Una prueba
   omitida por falta de fixture o dispositivo no debe informarse como aprobada.
3. Generar artefactos sin ofuscación, licencias, procedencia y hashes.
4. Verificar instalación y ejecución; conservar evidencia por versión.
5. Gestionar certificados y credenciales
   como secretos sin incluirlos en artefactos o logs públicos.
6. Publicar un borrador de versión para revisión. Subir a Store y hacer pública
   una versión son pasos posteriores, sujetos a la autorización del titular.

No se crea todavía un workflow que publique al hacer push.

## Canal Microsoft Store

Decisión del titular: producto de pago publicado como persona natural. El EULA
se adaptó a esa intención y sigue siendo un borrador jurídico. Pendientes:
correo público, precio, modalidad de cobro, soporte y ficha comercial.

La ruta candidata para cobrar la adquisición mediante la tienda es MSIX:
Microsoft documenta [precio y disponibilidad de MSIX](https://learn.microsoft.com/en-us/windows/apps/publish/publish-your-app/msix/price-and-availability).
El repositorio todavía no genera MSIX ni integra la validación comercial de
licencias. Ofuscar un ejecutable no acredita que el usuario lo haya comprado.
No se ha reservado una identidad de paquete ni creado una cuenta de publicación.

La [política 7.20, vigente desde el 22 de octubre de 2026](https://learn.microsoft.com/en-us/windows/apps/publish/store-policies)
debe verificarse antes de la entrega. Su sección 10.14 distingue la cuenta
Individual de Company según el carácter profesional/comercial y la identidad
presentada. Que el titular no haya constituido una sociedad no decide por sí
solo el tipo de cuenta aceptable; debe confirmarse en el registro o con soporte.

Para distribuir EXE/MSI mediante URL, la sección 10.2.9 exige firma y ejecución
silenciosa. El instalador actual no está firmado y exige un parámetro explícito
de EULA para despliegues silenciosos. Su pantalla interactiva no resuelve la
aceptación en una instalación silenciosa de Store. Hay que diseñar la entrega
previa de condiciones y aceptación correspondiente al canal elegido.

Antes de enviar a certificación hay que completar empaquetado, privacidad,
licencias de terceros, metadatos, pruebas, identidad y condiciones comerciales.
Esta rama prepara ese trabajo; no representa una app ya aprobada para Store.
