#!/usr/bin/env bash
set -euo pipefail

deb="${1:-}"
image="${2:-ubuntu:24.04}"
if [ -z "$deb" ]; then
	printf 'usage: scripts/test-deb-install.sh DEB [IMAGE]\n' >&2
	exit 2
fi
if [ ! -f "$deb" ]; then
	printf 'test-deb-install: missing %s\n' "$deb" >&2
	exit 1
fi
abs_deb="$(cd "$(dirname "$deb")" && pwd -P)/$(basename "$deb")"
docker run --rm -v "$abs_deb:/tmp/lyn.deb:ro" "$image" bash -lc '
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y --no-install-recommends /tmp/lyn.deb
test -x /usr/bin/lyn
test -f /usr/share/applications/lyn.desktop
test -f /etc/xdg/autostart/lyn.desktop
dpkg-query -W -f="\${Package} \${Version} \${Architecture}\n" lyn
'
