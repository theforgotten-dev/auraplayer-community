#!/usr/bin/env bash
#
# Build a native Debian 13 (trixie) .deb for Aura Music Player.
#
# Requirements (build host):
#   - Go toolchain satisfying go.mod (go 1.27.1 or newer)
#   - Gio build dependencies (X11/Wayland/EGL/GLES headers) and pkg-config
#   - dpkg-deb, dpkg-shlibdeps (dpkg-dev)
#
# Usage:
#   packaging/build-deb.sh [UPSTREAM_VERSION] [DEBIAN_REVISION]
#
# Example:
#   packaging/build-deb.sh 1.0.0 1
#
# The package is written to dist/aura-player_<version>_amd64.deb.
set -euo pipefail
umask 022

PKG=aura-player
ARCH=amd64
UPSTREAM_VERSION="${1:-1.0.0}"
DEBIAN_REVISION="${2:-1}"
VERSION="${UPSTREAM_VERSION}-${DEBIAN_REVISION}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PACKAGING_DIR="$REPO_ROOT/packaging"
DIST_DIR="$REPO_ROOT/dist"
STAGE_DIR="$DIST_DIR/deb"
PKGROOT="$STAGE_DIR/${PKG}_${VERSION}_${ARCH}"
BINARY="$DIST_DIR/$PKG"

if [[ ! "$UPSTREAM_VERSION" =~ ^[0-9][0-9A-Za-z.+:~-]*$ ]]; then
	echo "error: invalid upstream version '$UPSTREAM_VERSION'" >&2
	exit 1
fi
if [[ ! "$DEBIAN_REVISION" =~ ^[0-9][0-9A-Za-z.+~]*$ ]]; then
	echo "error: invalid Debian revision '$DEBIAN_REVISION'" >&2
	exit 1
fi

echo ">> Building $PKG $VERSION for linux/$ARCH"
mkdir -p "$DIST_DIR"
(
	cd "$REPO_ROOT"
	GOOS=linux GOARCH="$ARCH" CGO_ENABLED=1 \
		go build -trimpath -ldflags "-s -w" -o "$BINARY" .
)

echo ">> Staging package tree at $PKGROOT"
rm -rf "$PKGROOT"
install -d "$PKGROOT/DEBIAN"
install -d "$PKGROOT/usr/bin"
install -d "$PKGROOT/usr/share/applications"
install -d "$PKGROOT/usr/share/icons/hicolor/256x256/apps"
install -d "$PKGROOT/usr/share/man/man1"
install -d "$PKGROOT/usr/share/doc/$PKG"

install -m 0755 "$BINARY" "$PKGROOT/usr/bin/$PKG"
install -m 0644 "$PACKAGING_DIR/debian/aura-player.desktop" \
	"$PKGROOT/usr/share/applications/aura-player.desktop"
install -m 0644 "$REPO_ROOT/assets/aura-icon.png" \
	"$PKGROOT/usr/share/icons/hicolor/256x256/apps/$PKG.png"
install -m 0644 "$PACKAGING_DIR/debian/aura-player.1" \
	"$PKGROOT/usr/share/man/man1/aura-player.1"
install -m 0644 "$REPO_ROOT/LICENSE" "$PKGROOT/usr/share/doc/$PKG/copyright"
install -m 0644 "$REPO_ROOT/README.md" "$PKGROOT/usr/share/doc/$PKG/README.md"
gzip -9n -c "$PACKAGING_DIR/debian/changelog" \
	> "$PKGROOT/usr/share/doc/$PKG/changelog.Debian.gz"

echo ">> Resolving runtime dependencies (dpkg-shlibdeps)"
SHLIBS_DEPENDS=""
install -d "$PKGROOT/debian"
printf 'Source: %s\nPackage: %s\nArchitecture: %s\n' "$PKG" "$PKG" "$ARCH" \
	> "$PKGROOT/debian/control"
if SHLIBS_DEPENDS="$(cd "$PKGROOT" && dpkg-shlibdeps -O "usr/bin/$PKG" 2>/dev/null \
	| sed -n 's/^shlibs:Depends=//p')" && [[ -n "$SHLIBS_DEPENDS" ]]; then
	echo "   shlibs: $SHLIBS_DEPENDS"
else
	echo "   warning: dpkg-shlibdeps did not report dependencies; using fallback" >&2
	SHLIBS_DEPENDS="libc6"
fi
rm -rf "$PKGROOT/debian"

# ALSA is loaded at runtime via dlopen by the Oto engine, so it never shows up
# in the ELF NEEDED entries that dpkg-shlibdeps inspects.
EXTRA_DEPENDS="libasound2t64"
DEPENDS="$(printf '%s, %s' "$SHLIBS_DEPENDS" "$EXTRA_DEPENDS")"

cat > "$PKGROOT/DEBIAN/control" <<EOF
Package: $PKG
Version: $VERSION
Section: sound
Priority: optional
Architecture: $ARCH
Depends: $DEPENDS
Maintainer: Juan Pablo Muñoz Leiva <theforgotten-dev@users.noreply.github.com>
Homepage: https://github.com/theforgotten-dev/auraplayer
Description: Modular GPU-rendered audio player
 Aura Music Player is a modular audio player written in Go. It uses Oto as the
 PCM engine, go-mp3 for streaming MP3 decoding and Gio for a desktop interface
 rendered directly by the GPU. It does not use a WebView, HTML or JavaScript.
 .
 Supported formats include MP3, WAV, FLAC and Ogg Vorbis. Queues may mix
 formats, mono or stereo files and different sample rates.
EOF

DEB="$DIST_DIR/${PKG}_${VERSION}_${ARCH}.deb"
echo ">> Building $DEB"
dpkg-deb --build --root-owner-group -Zxz "$PKGROOT" "$DEB" >/dev/null

echo ">> Done: $DEB"
