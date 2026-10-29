> Documento histórico de la edición comercial heredada. No define la licencia de
> Community: rigen `LICENSE` (GPL-3.0-only) y `legal/COMMUNITY-LICENSE.md`.
> Las referencias a EULA propietario, pago o aceptación no se aplican a Community.

# Documentos legales de Aura Player

Estado: propuesta de trabajo, 24 de octubre de 2026. No es dictamen jurídico,
certificación de cumplimiento ni promesa de inmunidad. Los textos requieren
revisión profesional según la entidad que distribuya Aura, su modelo comercial,
los estados de EE. UU. donde opere y los demás países de destino.

## Archivos y alcance

- `legal/EULA.es.txt` y `legal/EULA.en.txt`: licencia de uso del binario, uso
  lícito, garantías, responsabilidad con excepciones, terceros y disputas.
- `legal/USO-RESPONSABLE.txt`: aviso breve bilingüe de uso y seguridad.
- `LICENSE`: conserva la reserva sobre el código del repositorio y reconoce
  expresamente la licencia limitada de los binarios oficiales bajo el EULA.

La identidad y el enlace de contacto se tomaron del LICENSE existente, sin
inventar empresa, domicilio, RUT, correo ni presencia en EE. UU. Hay que
confirmarlos y completar la identificación e información precontractual que
correspondan antes de una oferta comercial. El texto no crea una suscripción
ni fija su precio. El usuario confirmó distribución de pago mediante Microsoft
Store como persona natural. Sigue pendiente un correo público de soporte y
definir precio y modalidad de cobro. No se ha constituido una empresa por
medio de estos documentos ni se ha creado una cuenta en Partner Center.

El acuerdo distribuye obligaciones entre sus partes; no impide demandas ni
vincula a víctimas, titulares de derechos, autoridades o terceros ajenos.
No excluye responsabilidad penal o administrativa, ni convierte una función
ilícita o una conducta propia del proveedor en lícita. Un aviso antipiratería
no sustituye el análisis de las funciones reales y de su promoción.

## Criterio de redacción y fuentes consultadas

1. Chile: [Ley 19.496, texto refundido DFL 3](https://www.bcn.cl/leychile/navegar?idNorma=1160403),
   artículos 3, 4, 12 A, 16 y 17. Cuando la ley sea aplicable, deben respetarse
   derechos irrenunciables, información y acceso previo al contrato, aceptación
   inequívoca, disponibilidad de copia y controles de cláusulas abusivas.
   No se ofrece una exención absoluta ni un tope general de responsabilidad
   para consumidores. Debe revisarse además la legibilidad real del instalador;
   compilar no acredita el cumplimiento del tamaño exigido por el artículo 17.
2. Chile: [Código Civil](https://www.bcn.cl/leychile/navegar?idNorma=172986),
   artículo 1465. No se pretende perdonar anticipadamente el dolo futuro.
3. Chile: [Ley 17.336](https://www.bcn.cl/leychile/navegar?idNorma=28933),
   artículo 71 Ñ. Las restricciones sobre ingeniería inversa preservan las
   excepciones legales aplicables y los derechos de las licencias LGPL.
4. EE. UU.: [Berman v. Freedom Financial Network, Ninth Circuit, 2022](https://www.govinfo.gov/app/details/USCOURTS-ca9-20-16900).
   Como referencia judicial sobre formación de contratos electrónicos, motiva
   la presentación visible y la acción expresa de aceptar. No se extrapola
   como regla única de todos los estados ni garantía de exigibilidad.
5. EE. UU.: [15 USC 7001](https://www.govinfo.gov/content/pkg/USCODE-2024-title15/html/USCODE-2024-title15-chap96-subchapI-sec7001.htm).
   El formato electrónico no sustituye los demás requisitos legales. Cuando
   una información deba darse por escrito a consumidores, hay que revisar las
   condiciones específicas de entrega y consentimiento electrónico aplicables.
6. EE. UU.: [FTC, guía de garantías](https://www.ftc.gov/business-guidance/resources/businesspersons-guide-federal-warranty-law).
   Existen límites federales y estatales a los descargos. Hay que determinar
   si Magnuson-Moss resulta aplicable al producto y modalidad concreta; no se
   presume que cubra automáticamente todo software independiente. Garantías
   escritas y contratos de servicio requieren análisis separado.
7. Derechos de autor: [17 USC 1201](https://www.govinfo.gov/content/pkg/USCODE-2024-title17/html/USCODE-2024-title17-chap12-sec1201.htm).
   El uso lícito conserva las excepciones pertinentes; el acuerdo no autoriza
   elusión ilícita ni promete una excepción universal.

No se incluyeron arbitraje obligatorio, renuncia a acciones colectivas,
indemnidad ilimitada del usuario, foro extranjero exclusivo ni plazos propios
para acortar reclamos. Una versión B2B negociada podría requerir otras cláusulas,
pero no se mezcla con el contrato de adhesión para consumidores.

## Implementación del instalador

Inno Setup presenta el EULA completo en español o inglés antes del destino de
instalación, mediante `LicenseFile` por idioma. Ambos textos y la guía se
copian a `legal` para conservación. La documentación oficial de
[LicenseFile](https://jrsoftware.org/ishelp/topic_setup_licensefile.htm)
describe esta pantalla. No se presume aceptación por silencio.

La instalación silenciosa exige `/ACCEPTEULA=2026-10-24.2`; sin ese parámetro
se cancela. El parámetro es una declaración técnica de quien despliega y no
prueba identidad, representación, entrega previa ni consentimiento real de
usuarios finales. Un administrador debe tener autorización y documentar el
acuerdo pertinente antes del despliegue. Las pruebas automatizadas no equivalen
a aceptación contractual del usuario de este proyecto.

El portable incluye copias de los textos, pero no tiene todavía un mecanismo
propio de aceptación. No debe anunciarse como distribución con clickwrap ya
implementado. Para ese canal hace falta un flujo previo a la descarga o una
pantalla de primer inicio que muestre, permita guardar y aceptar/rechazar el
acuerdo. El EULA no afirma que la mera ejecución produzca consentimiento.

No se agrega telemetría ni se crea un registro remoto de aceptación. Para
evidencia contractual deben conservarse la versión exacta del acuerdo, hashes
del paquete y el flujo mostrado. Cualquier registro personal adicional exige
definir finalidad, retención, seguridad y aviso de privacidad. Un archivo local
editable no es por sí solo prueba concluyente de identidad o consentimiento.

## Antes de publicar

- Confirmar persona o empresa licenciante y canal de contacto efectivo;
  completar domicilio y datos de identificación cuando sean exigibles.
- Definir gratuidad o precio, pagos, reembolsos/retracto aplicables, soporte,
  actualizaciones prometidas y territorios concretos de comercialización.
- Revisar ambos idiomas con abogado en Chile y asesoría de EE. UU. según los
  estados y operación; no presentar el documento como cobertura mundial.
- Revisar la interfaz real de aceptación, accesibilidad y conservación de copia.
- Resolver el consentimiento del portable si se distribuye por ese canal.
- Inventariar tratamientos de datos antes de redactar una política de
  privacidad. No publicar afirmaciones de «no recopila datos» sin verificar
  también DLL, informes de fallos, distribución, soporte y futuras funciones.
- Completar la revisión de cumplimiento LGPL y fuentes correspondientes,
  demás dependencias, códecs/patentes según territorios y uso de marcas. Un
  EULA propio y enlaces a upstream no acreditan por sí solos ese cumplimiento.

Los instaladores 1.2.1 ya generados no se alteran ni adquieren estos términos
retroactivamente. Esta modificación del repositorio prepara futuros paquetes;
no significa que esos acuerdos se hayan aceptado ni que exista una nueva
versión pública revisada jurídicamente.
