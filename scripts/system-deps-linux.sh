#!/usr/bin/env bash
set -euo pipefail

have_pkg() {
	command -v pkg-config >/dev/null 2>&1 && pkg-config --exists "$1" 2>/dev/null
}

have_cc() {
	command -v cc >/dev/null 2>&1 || command -v gcc >/dev/null 2>&1
}

webkit_ok() {
	have_pkg webkit2gtk-4.1 || have_pkg webkit2gtk-4.0
}

appindicator_ok() {
	have_pkg ayatana-appindicator3-0.1 || have_pkg appindicator3-0.1
}

deps_satisfied() {
	have_cc && command -v pkg-config >/dev/null 2>&1 && have_pkg gtk+-3.0 && webkit_ok && appindicator_ok
}

if deps_satisfied; then
	exit 0
fi

echo "==> Installing Lyn build dependencies"

run_root() {
	if [ "$(id -u)" -eq 0 ]; then
		"$@"
	elif command -v sudo >/dev/null 2>&1; then
		sudo "$@"
	else
		echo "Missing build dependencies and neither root nor sudo is available." >&2
		echo "Install: a C compiler, pkg-config, GTK3, WebKit2GTK (4.1 or 4.0) and Ayatana AppIndicator development packages." >&2
		exit 1
	fi
}

if command -v apt-get >/dev/null 2>&1; then
	webkit_pkg=libwebkit2gtk-4.1-dev
	apt-cache show libwebkit2gtk-4.1-dev >/dev/null 2>&1 || webkit_pkg=libwebkit2gtk-4.0-dev
	run_root apt-get update
	run_root apt-get install -y build-essential pkg-config libgtk-3-dev "$webkit_pkg" libayatana-appindicator3-dev
elif command -v dnf >/dev/null 2>&1; then
	run_root dnf install -y gcc pkgconf-pkg-config gtk3-devel webkit2gtk4.1-devel libayatana-appindicator-gtk3-devel
elif command -v pacman >/dev/null 2>&1; then
	run_root pacman -S --needed --noconfirm base-devel pkgconf gtk3 webkit2gtk-4.1 libayatana-appindicator
else
	echo "Unsupported package manager. Install GTK3, WebKit2GTK and Ayatana AppIndicator dev packages manually." >&2
	exit 1
fi

if ! deps_satisfied; then
	echo "Build dependencies still missing after install attempt." >&2
	exit 1
fi
