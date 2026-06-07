#!/usr/bin/env bash
set -euo pipefail

usage() {
	printf '%s\n' 'usage: scripts/package-deb.sh ROOT VERSION ARCH OUTPUT'
}

deb_arch() {
	case "$1" in
	x64 | amd64 | x86_64) printf 'amd64' ;;
	arm64 | aarch64) printf 'arm64' ;;
	*) printf '%s' "$1" ;;
	esac
}

root="${1:-}"
version="${2:-}"
arch="${3:-}"
output="${4:-}"
if [ -z "$root" ] || [ -z "$version" ] || [ -z "$arch" ] || [ -z "$output" ]; then
	usage >&2
	exit 2
fi
binary="$root/build/bin/lyn"
icon="$root/build/appicon.png"
if [ ! -f "$binary" ]; then
	printf 'package-deb: missing %s\n' "$binary" >&2
	exit 1
fi
if [ ! -f "$icon" ]; then
	printf 'package-deb: missing %s\n' "$icon" >&2
	exit 1
fi
command -v dpkg-deb >/dev/null 2>&1 || {
	printf 'package-deb: dpkg-deb is required\n' >&2
	exit 1
}
stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT
pkg="$stage/pkg"
mkdir -p "$pkg/DEBIAN" "$pkg/usr/bin" "$pkg/usr/share/applications" "$pkg/usr/share/pixmaps" "$pkg/etc/xdg/autostart"
install -m 0755 "$binary" "$pkg/usr/bin/lyn"
install -m 0644 "$icon" "$pkg/usr/share/pixmaps/lyn.png"
cat >"$pkg/usr/share/applications/lyn.desktop" <<'DESKTOP'
[Desktop Entry]
Type=Application
Name=Lyn
Comment=Project launcher
Exec=/usr/bin/lyn
Icon=lyn
Terminal=false
Categories=Utility;
StartupNotify=false
DESKTOP
cat >"$pkg/etc/xdg/autostart/lyn.desktop" <<'DESKTOP'
[Desktop Entry]
Type=Application
Name=Lyn
Comment=Project launcher
Exec=/usr/bin/lyn --start-hidden
Icon=lyn
Terminal=false
Categories=Utility;
StartupNotify=false
X-GNOME-Autostart-enabled=true
DESKTOP
installed_size="$(du -sk "$pkg/usr" "$pkg/etc" | awk '{sum += $1} END {print sum}')"
cat >"$pkg/DEBIAN/control" <<CONTROL
Package: lyn
Version: $version
Section: utils
Priority: optional
Architecture: $(deb_arch "$arch")
Maintainer: lyn-tools
Installed-Size: $installed_size
Depends: libgtk-3-0, libwebkit2gtk-4.1-0 | libwebkit2gtk-4.0-37, libayatana-appindicator3-1 | libappindicator3-1
Homepage: https://github.com/asolopovas/lyn-tools
Description: High-performance desktop project launcher
 Lyn is a Wails desktop launcher for projects, applications, and VS Code workspaces.
CONTROL
mkdir -p "$(dirname "$output")"
dpkg-deb --build --root-owner-group "$pkg" "$output" >/dev/null
printf '%s\n' "$output"
