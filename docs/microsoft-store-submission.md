# Subir Aura Player a Microsoft Store

Estado del proyecto: rama `feature/installer-releases`. Hay instalador EXE y
portable 1.2.1; todavía no hay paquete MSIX validado para Store. El EXE actual
no está firmado. No se ha iniciado sesión, reservado un nombre ni enviado
una aplicación desde este repositorio.

## Primer paso en la cuenta del titular

1. Entrar en https://storedeveloper.microsoft.com/ y abrir el registro de
   desarrollador para Windows. Completar personalmente identidad, país y
   aceptación de contratos. Confirmar el tipo de cuenta: la política 10.14
   distingue actividad comercial/profesional de proyectos individuales; no
   elegir automáticamente Individual solo por no tener una sociedad.
2. En Partner Center, crear un producto para **MSIX o PWA**, siguiendo la ruta
   MSIX, y reservar el nombre comercial disponible. No asumir que "Aura Player"
   esté libre ni que reservarlo conceda derechos de marca.
3. Abrir la identidad del producto y copiar los valores exactos de
   `Package/Identity/Name`, `Package/Identity/Publisher` y
   `Package/Properties/PublisherDisplayName` para preparar el manifiesto.
   Son identificadores de empaquetado: no hace falta compartir contraseña,
   documentos de identidad, datos bancarios, claves o códigos de autenticación.

## Preparación local posterior

Con esa identidad se podrá generar un manifiesto y paquete MSIX para el
ejecutable Win32 de Aura, incluir libmpv y licencias, iconos adecuados y las
declaraciones de capacidades y asociaciones que correspondan. Hay que validar
instalación, reproducción, subtítulos, preferencias, apertura de archivos,
actualización y desinstalación en el contexto del paquete.

El SDK de Windows está presente en este equipo. Eso no significa que ya exista
un MSIX preparado ni que sus pruebas hayan pasado. No subir el EXE a la sección
MSIX ni cambiar la extensión de un ZIP a MSIX.

Para la distribución de un MSIX por Store, Microsoft firma el paquete durante
la publicación tras la certificación. La ruta EXE/MSI por URL exige firma propia
y no recibe automáticamente esa firma de Microsoft.

## Borrador de envío en Partner Center

1. Crear una nueva entrega del producto reservado.
2. Seleccionar mercados y precio de pago. Definir si habrá prueba y su alcance;
   no activar una prueba sin implementar y verificar su comportamiento.
3. Completar propiedades, categoría y cuestionario de clasificación de edad.
4. Subir el paquete MSIX de identidad correcta y resolver las validaciones.
5. Completar las fichas en español/inglés: descripción veraz, requisitos,
   capturas propias con contenido autorizado, iconos y contacto de soporte.
6. Proporcionar la política de privacidad aplicable y las condiciones de
   licencia antes de la compra. Los EULA actuales son borradores; la pantalla
   del instalador Inno no se ejecuta en una instalación MSIX y no acredita
   consentimiento de un cliente de Store.
7. Completar la configuración fiscal y de pagos requerida para cobrar. El
   titular debe proporcionar esos datos directamente a Microsoft.
8. Añadir notas para certificación y pruebas reproducibles de las funciones.
   Revisar el envío completo antes de someterlo a certificación/publicación.

Siguen pendientes el correo público, precio, revisión de privacidad/licencias,
identidad del producto y empaquetado MSIX. Ofuscación y EULA no sustituyen esos
requisitos ni la verificación de compra que se decida implementar.

## Fuentes oficiales

- Inicio: https://storedeveloper.microsoft.com/
- Publicación: https://learn.microsoft.com/windows/apps/package-and-deploy/publish-first-app
- Precio MSIX: https://learn.microsoft.com/windows/apps/publish/publish-your-app/msix/price-and-availability
- Requisitos de paquetes: https://learn.microsoft.com/windows/apps/publish/publish-your-app/package-version-numbering
- Firma y canales: https://learn.microsoft.com/windows/apps/package-and-deploy/code-signing-options
- Políticas: https://learn.microsoft.com/windows/apps/publish/store-policies

La navegación puede variar según el idioma y la cuenta. Ninguna de estas
acciones de cuenta se ha realizado automáticamente.
