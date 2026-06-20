#!/usr/bin/env bash
set -euo pipefail

root="${1:-$(pwd)}"
bin_dir="${2:-build/bin}"
cd "$root"

bash "$root/scripts/system-deps-linux.sh"

mkdir -p "$bin_dir"
webkit_tags=""
pkg-config --exists webkit2gtk-4.0 || webkit_tags="webkit2_41"
if command -v wails >/dev/null 2>&1; then
	wails build -m -nosyncgomod -s -tags "$webkit_tags"
else
	tags="desktop,production"
	test -z "$webkit_tags" || tags="$tags,$webkit_tags"
	go build -tags "$tags" -ldflags="-w -s" -o "$bin_dir/lyn" .
fi
