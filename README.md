# Aura Player Community

Reproductor multimedia libre para Windows 10 y 11 x64. Reproduce audio y video
locales, subtítulos y radio online con Radio Browser o Icecast. Incluye cola
compartida, ecualizador y visualizador de espectro con colores RGB configurables.

[Sitio oficial](https://theforgotten-dev.github.io/auraplayer-community/) ·
[Descargar para Windows](https://theforgotten-dev.github.io/auraplayer-community/descargar.html) ·
[Formatos compatibles](https://theforgotten-dev.github.io/auraplayer-community/formatos.html) ·
[Radio online](https://theforgotten-dev.github.io/auraplayer-community/radio-online.html)

## Desarrollo en Windows

Requiere la versión de Go indicada en `go.mod`.

```powershell
go mod download
go test ./...
go vet ./...
go run .
```

Para el motor multimedia, descarga y verifica el runtime fijado por el proyecto:

```powershell
./scripts/prepare-release-runtime.ps1
$env:AURA_MPV_DLL = (Resolve-Path build/release-runtime/libmpv-2.dll).Path
go run .
```

El empaquetado de Windows requiere Inno Setup 6. Consulta el script
[build-installer.ps1](installer/build-installer.ps1) y el workflow de release.
Los instaladores actuales no tienen firma digital.

## Seguridad

Consulta [SECURITY.md](SECURITY.md) para reportar vulnerabilidades de forma
privada. Las comprobaciones automáticas incluyen tests, vet, govulncheck,
Gitleaks y CodeQL. Las dependencias se revisan mediante Dependabot.

La carpeta `docs/` se conserva únicamente de forma local, sin seguimiento.

## Licencia

Copyright (c) 2026 Juan Pablo Muñoz Leiva (theforgotten-dev).

El código y los materiales originales de Community se distribuyen bajo
**GPL-3.0-only**. Puedes usar, estudiar, modificar y redistribuir el proyecto
cumpliendo sus condiciones. Consulta [LICENSE](LICENSE) y
[COMMUNITY-LICENSE.md](legal/COMMUNITY-LICENSE.md).
Las dependencias y materiales de terceros conservan sus licencias.

Community y la edición comercial mantienen repositorios y distribución
separados. Este repositorio no configura una sincronización automática.