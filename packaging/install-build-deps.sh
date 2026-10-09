#!/usr/bin/env bash
#
# Install the build dependencies required to compile Aura Music Player on
# Debian 13 (trixie) amd64. Run as root.
#
# These are build-time (-dev) packages. They are NOT the runtime dependencies
# declared by the generated .deb; see packaging/build-deb.sh.
set -euo pipefail

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
	echo "Please run as root: sudo $0" >&2
	exit 1
fi

apt-get update
apt-get install -y --no-install-recommends \
	build-essential pkg-config git ca-certificates \
	libx11-dev libx11-xcb-dev libxkbcommon-dev libxkbcommon-x11-dev \
	libwayland-dev libegl-dev libgles-dev libvulkan-dev \
	libxcursor-dev libxfixes-dev libxrandr-dev libxinerama-dev \
	libxi-dev libxext-dev libxrender-dev \
	libasound2-dev
