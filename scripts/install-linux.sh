#!/usr/bin/env bash
set -euo pipefail

cleanup_dir=""
trap 'if [ -n "${cleanup_dir:-}" ]; then rm -rf "$cleanup_dir"; fi' EXIT

command_path() {
	command -v "$1" 2>/dev/null || true
}

canonical_path() {
	local path="$1"
	if command -v realpath >/dev/null 2>&1; then
		realpath -m "$path" 2>/dev/null && return
	fi
	if [ -e "$path" ]; then
		(cd "$(dirname "$path")" 2>/dev/null && printf '%s/%s\n' "$(pwd -P)" "$(basename "$path")") && return
	fi
	printf '%s\n' "$path"
}

is_inside() {
	local path="$1"
	local root="$2"
	local full_path
	local full_root
	full_path="$(canonical_path "$path")" || return 1
	full_root="$(canonical_path "$root")" || return 1
	case "$full_path" in
	"$full_root" | "$full_root"/*) return 0 ;;
	*) return 1 ;;
	esac
}

install_target() {
	local root="$1"
	local existing
	existing="$(command_path lyn)"
	if [ -n "$existing" ] && ! is_inside "$existing" "$root"; then
		printf '%s\n' "$existing"
		return
	fi
	printf '%s\n' "$HOME/.local/bin/lyn"
}

stop_target() {
	local target="$1"
	local full_target
	local exe
	local running
	local pid
	full_target="$(canonical_path "$target")"
	[ -d /proc ] || return 0
	for exe in /proc/[0-9]*/exe; do
		[ -e "$exe" ] || continue
		running="$(readlink "$exe" 2>/dev/null || true)"
		if [ "$running" = "$full_target" ]; then
			pid="${exe#/proc/}"
			pid="${pid%/exe}"
			kill "$pid" 2>/dev/null || true
		fi
	done
}

desktop_exec_value() {
	local value="$1"
	value="${value//\\/\\\\}"
	value="${value//\"/\\\"}"
	printf '"%s"' "$value"
}

desktop_entry() {
	local target="$1"
	local icon="$2"
	local exec_value
	exec_value="$(desktop_exec_value "$target")"
	printf '%s\n' \
		'[Desktop Entry]' \
		'Type=Application' \
		'Name=Lyn' \
		'Comment=Project launcher' \
		"Exec=$exec_value" \
		"Icon=$icon" \
		'Terminal=false' \
		'Categories=Utility;' \
		'StartupNotify=false' \
		'X-GNOME-Autostart-enabled=true' \
		''
}

config_file() {
	if [ -n "${XDG_CONFIG_HOME:-}" ]; then
		printf '%s\n' "$XDG_CONFIG_HOME/lyn/lyn.json"
		return
	fi
	printf '%s\n' "$HOME/.config/lyn/lyn.json"
}

data_home() {
	if [ -n "${XDG_DATA_HOME:-}" ]; then
		printf '%s\n' "$XDG_DATA_HOME"
		return
	fi
	printf '%s\n' "$HOME/.local/share"
}

data_dir() {
	printf '%s\n' "$(data_home)/applications"
}

icon_file() {
	printf '%s\n' "$(data_home)/icons/lyn.png"
}

install_binary() {
	local source="$1"
	local target="$2"
	local target_dir
	target_dir="$(dirname "$target")"
	if [ ! -d "$target_dir" ]; then
		if ! mkdir -p "$target_dir" 2>/dev/null; then
			if command -v sudo >/dev/null 2>&1; then
				printf 'Requesting sudo to create %s\n' "$target_dir"
				sudo mkdir -p "$target_dir"
			else
				printf 'Cannot create %s and sudo is not available\n' "$target_dir" >&2
				exit 1
			fi
		fi
	fi
	if [ -w "$target_dir" ]; then
		install -m 0755 "$source" "$target"
		return
	fi
	if command -v sudo >/dev/null 2>&1; then
		printf 'Requesting sudo to install Lyn to %s\n' "$target"
		sudo install -m 0755 "$source" "$target"
		return
	fi
	printf 'Cannot write to %s and sudo is not available\n' "$target_dir" >&2
	exit 1
}

install_icon() {
	local source="$1"
	local target="$2"
	local target_dir
	target_dir="$(dirname "$target")"
	mkdir -p "$target_dir"
	install -m 0644 "$source" "$target"
}

set_startup_config() {
	local config="$1"
	if ! command -v go >/dev/null 2>&1; then
		printf 'go is required to update %s\n' "$config" >&2
		exit 1
	fi
	cleanup_dir="$(mktemp -d)"
	cat >"$cleanup_dir/update-config.go" <<'GO'
package main

import (
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"
)

func main() {
    if len(os.Args) != 2 {
        fatal(errors.New("config path is required"))
    }
    path := os.Args[1]
    cfg := map[string]any{}
    data, err := os.ReadFile(path)
    if err != nil && !errors.Is(err, os.ErrNotExist) {
        fatal(err)
    }
    if err == nil && strings.TrimSpace(string(data)) != "" {
        if err := json.Unmarshal(data, &cfg); err != nil {
            fatal(err)
        }
    }
    startup, _ := cfg["startup"].(map[string]any)
    if startup == nil {
        startup = map[string]any{}
    }
    startup["enabled"] = true
    startup["startHidden"] = true
    cfg["startup"] = startup
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        fatal(err)
    }
    out, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        fatal(err)
    }
    out = append(out, '\n')
    if err := os.WriteFile(path, out, 0o644); err != nil {
        fatal(err)
    }
}

func fatal(err error) {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
}
GO
	go run "$cleanup_dir/update-config.go" "$config"
	rm -rf "$cleanup_dir"
	cleanup_dir=""
}

root="${1:-}"
if [ -z "$root" ]; then
	printf 'repository root is required\n' >&2
	exit 1
fi
source_path="$root/build/bin/lyn"
icon_source="$root/build/appicon.png"
if [ ! -f "$source_path" ]; then
	printf 'Built executable not found at %s\n' "$source_path" >&2
	exit 1
fi
if [ ! -f "$icon_source" ]; then
	printf 'App icon not found at %s\n' "$icon_source" >&2
	exit 1
fi
if [ -z "${HOME:-}" ]; then
	printf 'HOME is not set\n' >&2
	exit 1
fi
target="$(install_target "$root")"
icon="$(icon_file)"
stop_target "$target"
install_binary "$source_path" "$target"
install_icon "$icon_source" "$icon"
desktop_dir="$(data_dir)"
mkdir -p "$desktop_dir"
desktop="$desktop_dir/lyn.desktop"
desktop_entry "$target" "$icon" >"$desktop"
autostart_dir="${XDG_CONFIG_HOME:-$HOME/.config}/autostart"
autostart="$autostart_dir/lyn.desktop"
mkdir -p "$autostart_dir"
cp "$desktop" "$autostart"
set_startup_config "$(config_file)"
if command -v update-desktop-database >/dev/null 2>&1; then
	update-desktop-database "$desktop_dir" >/dev/null 2>&1 || true
fi
printf 'Installed Lyn to %s\n' "$target"
printf 'Enabled Lyn startup with hidden background launch\n'
