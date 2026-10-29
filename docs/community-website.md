# Sitio de Aura Player Community

Dirección: https://theforgotten-dev.github.io/auraplayer-community/

## Mantenimiento

El sitio estático está en `site/`, sin dependencias de frontend ni servicios de terceros. Cambiarlo mediante ramas desde `dev` y PR hacia `dev`. El workflow `.github/workflows/pages.yml` publica únicamente `site/` al integrar cambios; no publica los documentos locales de continuidad ni compila el reproductor.

GitHub Pages debe tener `build_type: workflow`. El despliegue usa el entorno `github-pages`, permisos de Pages y OIDC limitados al job de publicación. No requiere secretos personales.

## SEO y contenido

La página tiene título, descripción, canonical, Open Graph, tarjeta social, datos estructurados WebSite y sitemap. Los materiales originales están bajo GPL-3.0-only. No se declaran calificaciones, descargas ni firma digital como hechos actuales. Actualizar el estado al publicar la primera versión.

La imagen procede de `assets/store/EN/aura-screenshot-audio-1366x768.png` y se identifica como interfaz de la base, no como captura de una versión Community ya publicada. El icono procede de `assets/aura-icon.png`.

No se incluye `robots.txt`: en un sitio de proyecto los robots consultan el archivo en la raíz del host (`theforgotten-dev.github.io/robots.txt`), fuera de la ruta de este repositorio.

Para Google Search Console, agregar una propiedad de prefijo de URL con la dirección completa, completar la verificación de propiedad en la cuenta del titular y enviar `sitemap.xml`. La cuenta y el código de verificación deben proceder del titular. Esto no garantiza indexación ni posición. No se activan campañas SEM.

## Validación manual

Revisar el sitio en escritorio y móvil, navegación por teclado, enlaces a GitHub y metadatos. Verificar el resultado del workflow y respuesta HTTP del sitio tras cada publicación. El sitio no usa cookies, formularios ni scripts de analítica.
