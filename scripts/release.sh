#!/usr/bin/env bash
set -euo pipefail

usage() {
	cat <<'EOF'
usage:
  just bump [patch|minor|major|X.Y.Z]
  just package-deb
  just test-deb-install [path/to/lyn.deb]
  just publish-assets [tag]
  just release [--bump patch|minor|major|X.Y.Z] [--force] [--no-push] [--dry-run]
  just release-patch [--force]
  just release-minor [--force]
  just release-major [--force]
EOF
}

current_version() {
	if [ -f VERSION ]; then
		tr -d '[:space:]' <VERSION
		return
	fi
	node -e "const j=require('./wails.json'); process.stdout.write(j.info?.productVersion || '0.1.0')"
}

valid_version() {
	[[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

next_version() {
	local current="${1#v}"
	local bump="$2"
	local major
	local minor
	local patch
	IFS=. read -r major minor patch <<<"$current"
	case "$bump" in
	patch) printf '%s.%s.%s\n' "$major" "$minor" "$((patch + 1))" ;;
	minor) printf '%s.%s.0\n' "$major" "$((minor + 1))" ;;
	major) printf '%s.0.0\n' "$((major + 1))" ;;
	v[0-9]*.[0-9]*.[0-9]*) printf '%s\n' "${bump#v}" ;;
	[0-9]*.[0-9]*.[0-9]*) printf '%s\n' "$bump" ;;
	*)
		printf 'release: invalid bump %s\n' "$bump" >&2
		exit 2
		;;
	esac
}

sync_version_files() {
	local old_version="$1"
	local new_version="$2"
	node - "$old_version" "$new_version" <<'JS'
const fs = require('node:fs');
const [oldVersion, newVersion] = process.argv.slice(2);
fs.writeFileSync('VERSION', `${newVersion}\n`);
const wails = JSON.parse(fs.readFileSync('wails.json', 'utf8'));
wails.info ??= {};
wails.info.productVersion = newVersion;
fs.writeFileSync('wails.json', `${JSON.stringify(wails, null, 2)}\n`);
if (fs.existsSync('README.md')) {
  let text = fs.readFileSync('README.md', 'utf8');
  text = text.split(oldVersion).join(newVersion);
  fs.writeFileSync('README.md', text);
}
JS
}

ensure_clean() {
	git diff --quiet || {
		printf 'release: tracked files changed; commit or stash first\n' >&2
		exit 1
	}
	git diff --cached --quiet || {
		printf 'release: staged changes exist; commit or stash first\n' >&2
		exit 1
	}
}

ensure_release_changes() {
	local changed
	changed="$(git diff --name-only)"
	while IFS= read -r path; do
		[ -n "$path" ] || continue
		case "$path" in
		VERSION | README.md | wails.json | frontend/dist/*) ;;
		*)
			printf 'release: unexpected tracked change after build: %s\n' "$path" >&2
			exit 1
			;;
		esac
	done <<<"$changed"
}

platform_name() {
	case "$(uname -s)" in
	MINGW* | MSYS* | CYGWIN*) printf 'windows' ;;
	Linux*) printf 'linux' ;;
	Darwin*) printf 'macos' ;;
	*) uname -s | tr '[:upper:]' '[:lower:]' ;;
	esac
}

arch_name() {
	case "$(uname -m)" in
	x86_64 | amd64) printf 'x64' ;;
	aarch64 | arm64) printf 'arm64' ;;
	*) uname -m ;;
	esac
}

prepare_release_dir() {
	local tag="$1"
	local out_dir="releases/$tag"
	rm -rf "$out_dir"
	mkdir -p "$out_dir"
	printf '%s\n' "$out_dir"
}

write_checksums() {
	local out_dir="$1"
	local files=()
	local pattern
	shopt -s nullglob
	for pattern in "$out_dir"/*.exe "$out_dir"/*.deb; do
		files+=("$(basename "$pattern")")
	done
	shopt -u nullglob
	[ "${#files[@]}" -gt 0 ] || return 0
	if command -v sha256sum >/dev/null 2>&1; then
		(cd "$out_dir" && sha256sum "${files[@]}" >SHA256SUMS)
	else
		(cd "$out_dir" && shasum -a 256 "${files[@]}" >SHA256SUMS)
	fi
}

package_deb() {
	local tag="$1"
	local version="$2"
	local arch="$3"
	local out_dir="$4"
	local output="$out_dir/lyn-$tag-linux-$(arch_name).deb"
	bash "scripts/package-deb.sh" "$PWD" "$version" "$arch" "$output"
}

package_native_installers() {
	local tag="$1"
	local version="$2"
	local platform="$3"
	local arch="$4"
	local out_dir="$5"
	case "$platform" in
	linux) package_deb "$tag" "$version" "$arch" "$out_dir" ;;
	esac
	write_checksums "$out_dir"
}

upload_release() {
	local tag="$1"
	local out_dir="$2"
	local force="$3"
	command -v gh >/dev/null 2>&1 || {
		printf 'release: gh CLI is required to upload\n' >&2
		exit 1
	}
	git push origin HEAD
	if [ "$force" = true ]; then
		git push origin "refs/tags/$tag:refs/tags/$tag" --force
		gh release delete "$tag" --yes >/dev/null 2>&1 || true
	else
		git push origin "$tag"
		if gh release view "$tag" >/dev/null 2>&1; then
			printf 'release: GitHub release exists: %s; use --force to replace\n' "$tag" >&2
			exit 1
		fi
	fi
	gh release create "$tag" "$out_dir"/* --title "Lyn $tag" --notes "Lyn $tag"
}

publish_assets_command() {
	local tag="${1:-}"
	local version
	local out_dir
	if [ -z "$tag" ]; then
		version="$(current_version)"
		tag="v${version#v}"
	fi
	out_dir="releases/$tag"
	[ -d "$out_dir" ] || {
		printf 'release: missing asset directory %s\n' "$out_dir" >&2
		exit 1
	}
	command -v gh >/dev/null 2>&1 || {
		printf 'release: gh CLI is required to upload\n' >&2
		exit 1
	}
	while IFS= read -r asset; do
		[ -n "$asset" ] || continue
		case "$asset" in
		*.tar.gz | SHA256SUMS) gh release delete-asset "$tag" "$asset" --yes ;;
		esac
	done < <(gh release view "$tag" --json assets --jq '.assets[].name')
	gh release upload "$tag" "$out_dir"/*.deb "$out_dir"/SHA256SUMS --clobber
	gh release edit "$tag" --latest 2>/dev/null || true
}

bump_command() {
	local bump="${1:-patch}"
	local current
	local next
	current="$(current_version)"
	valid_version "${current#v}" || {
		printf 'release: invalid current version %s\n' "$current" >&2
		exit 1
	}
	next="$(next_version "$current" "$bump")"
	valid_version "$next" || {
		printf 'release: invalid next version %s\n' "$next" >&2
		exit 1
	}
	sync_version_files "${current#v}" "$next"
	printf '%s\n' "$next"
}

release_command() {
	local bump=""
	local push=true
	local force=false
	local dry_run=false
	local current
	local next
	local tag
	local platform
	local arch
	local out_dir
	while [ $# -gt 0 ]; do
		case "$1" in
		--bump)
			bump="${2:-patch}"
			if [ $# -ge 2 ] && [[ "$2" != --* ]]; then shift 2; else
				bump="patch"
				shift
			fi
			;;
		--bump=*)
			bump="${1#--bump=}"
			[ -n "$bump" ] || bump="patch"
			shift
			;;
		--force)
			force=true
			shift
			;;
		--no-push)
			push=false
			shift
			;;
		--push)
			push=true
			shift
			;;
		--dry-run)
			dry_run=true
			shift
			;;
		-h | --help)
			usage
			exit 0
			;;
		*)
			printf 'release: unknown argument %s\n' "$1" >&2
			usage >&2
			exit 2
			;;
		esac
	done
	current="$(current_version)"
	valid_version "${current#v}" || {
		printf 'release: invalid current version %s\n' "$current" >&2
		exit 1
	}
	next="${current#v}"
	if [ -n "$bump" ]; then
		next="$(next_version "$current" "$bump")"
	fi
	tag="v$next"
	platform="$(platform_name)"
	arch="$(arch_name)"
	if [ "$dry_run" = true ]; then
		printf 'release: version=%s tag=%s platform=%s arch=%s push=%s force=%s\n' "$next" "$tag" "$platform" "$arch" "$push" "$force"
		exit 0
	fi
	ensure_clean
	if [ -n "$bump" ]; then
		sync_version_files "${current#v}" "$next"
	fi
	just check
	just build
	out_dir="$(prepare_release_dir "$tag")"
	package_native_installers "$tag" "$next" "$platform" "$arch" "$out_dir"
	if [ -n "$bump" ]; then
		ensure_release_changes
		git add VERSION README.md wails.json frontend/dist
		git commit -m "Release $tag"
	elif ! git diff --quiet; then
		printf 'release: build changed tracked files; rerun with --bump or commit build output first\n' >&2
		git diff --name-only >&2
		exit 1
	fi
	if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
		if [ "$force" = true ]; then
			git tag -d "$tag" >/dev/null
		else
			printf 'release: tag exists: %s; use --force to move it\n' "$tag" >&2
			exit 1
		fi
	fi
	git tag "$tag"
	if [ "$push" = true ]; then
		upload_release "$tag" "$out_dir" "$force"
	fi
	printf 'release: %s ready in %s\n' "$tag" "$out_dir"
}

package_deb_command() {
	local version
	local tag
	local out_dir
	version="$(current_version)"
	tag="v${version#v}"
	out_dir="releases/$tag"
	mkdir -p "$out_dir"
	package_deb "$tag" "${version#v}" "$(arch_name)" "$out_dir"
	write_checksums "$out_dir"
}

test_deb_install_command() {
	local deb="${1:-}"
	local version
	local tag
	if [ -z "$deb" ]; then
		version="$(current_version)"
		tag="v${version#v}"
		deb="releases/$tag/lyn-$tag-linux-$(arch_name).deb"
	fi
	bash "scripts/test-deb-install.sh" "$deb"
}

cmd="${1:-}"
case "$cmd" in
bump)
	shift
	bump_command "$@"
	;;
package-deb)
	shift
	package_deb_command "$@"
	;;
test-deb-install)
	shift
	test_deb_install_command "$@"
	;;
publish-assets)
	shift
	publish_assets_command "$@"
	;;
release)
	shift
	release_command "$@"
	;;
-h | --help | help | '') usage ;;
*)
	printf 'unknown command: %s\n' "$cmd" >&2
	usage >&2
	exit 2
	;;
esac
