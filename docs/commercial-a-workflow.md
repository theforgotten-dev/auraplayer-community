# Aura Player: versión A comercial

El repositorio privado https://github.com/theforgotten-dev/auraplayer es la fuente de la versión A comercial. Su licencia y los acuerdos de distribución están definidos en LICENSE y en la documentación legal incorporada mediante el PR #1. La denominación comercial no acredita firma, aprobación de Store ni una publicación.

## Ramas y revisiones

- `dev`: integración y destino de los PR; rama predeterminada del repositorio.
- `feature/*`, `fix/*`, `chore/*`, `docs/*` y `packaging/*`: trabajo acotado con PR hacia `dev`.
- `release/*`: preparación de una versión desde un commit validado de `dev`.
- `hotfix/*`: corrección de una versión publicada, con reintegración mediante PR.
- `linux`, `desktop` y `master`: referencias históricas conservadas; no son destinos de nuevas integraciones.

No se eliminan ramas históricas ni se reescribe su historial. Las ramas de funcionalidades ya contenidas en `dev` no necesitan PR adicionales. Para publicar una versión se deben registrar el commit, la versión, las pruebas, los hashes y el estado de firma e instalación.

## PR pendientes hacia dev

| PR | Alcance | Estado de validación |
| --- | --- | --- |
| [#1](https://github.com/theforgotten-dev/auraplayer/pull/1) | Instaladores Windows, recursos Store, textos legales y referencias | Pruebas Go, vet y compilación en Windows; no equivale a publicación en Store |
| [#2](https://github.com/theforgotten-dev/auraplayer/pull/2) | Empaquetado Debian 13 | Borrador; requiere construcción, instalación y reproducción en Debian |

El ecualizador, preferencias, radio y el historial de las otras ramas de funcionalidades ya están contenidos en `dev`. Android conserva una rama de exploración y requiere una implementación y validación propias antes de anunciar soporte.

## Remotos del checkout comercial

`origin` apunta a `https://github.com/theforgotten-dev/auraplayer.git`. Todas las ramas comerciales deben seguir sus equivalentes en `origin`; `git push` publica solo la rama actual mediante `push.default=simple` y `remote.pushDefault=origin`.

`community` apunta al repositorio público independiente `https://github.com/theforgotten-dev/auraplayer-community.git`. No existe sincronización automática. Una publicación pública requiere un alcance explícito y revisión de los archivos e historial que se van a enviar. Los cambios comerciales no se publican allí como parte de la integración hacia `dev`.

```powershell
git fetch origin --prune
git switch dev
git pull --ff-only origin dev
git switch -c feature/nombre
git push -u origin feature/nombre
gh pr create --repo theforgotten-dev/auraplayer --base dev
```

Antes de solicitar revisión de cambios Go: `go test ./...`, `go vet ./...` y `go build ./...`. Los cambios de empaquetado también necesitan la prueba de instalación y ejecución en su plataforma. Crear un PR no implica fusionarlo ni publicar una versión.