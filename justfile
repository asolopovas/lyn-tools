set shell := ["bash", "-uc"]
set windows-shell := ["pwsh", "-NoLogo", "-NoProfile", "-Command"]

frontend := "frontend"
build-bin := "build/bin"

[default]
[windows]
help:
    @Write-Output @('LYN JUSTFILE(1)', '', 'NAME', '  just - show project command help', '', 'SYNOPSIS', '  just <command> [arguments]', '', 'QUALITY', '  just check                    Run all Go and frontend quality gates', '  just go-quality               Run gofmt check, tests, vet, and staticcheck', '  just frontend-quality         Run frontend formatting, lint, and type checks', '  just lint                     Run Go quality and frontend lint', '  just format                   Format Go and frontend files', '', 'BUILD AND RUN', '  just build                    Build the desktop app', '  just dev                      Start Wails development mode', '  just start                    Alias for dev', '  just scan                     Run the launcher from source', '  just install                  Install deps, build, and install the current working tree', '  just install-main             Sync clean main, install deps, build, and install', '', 'RELEASE', '  just bump [level]             Bump VERSION; level defaults to patch', '  just release [args...]        Package and publish a release', '  just release-patch [args...]  Bump patch, then release', '  just release-minor [args...]  Bump minor, then release', '  just release-major [args...]  Bump major, then release', '', 'MAINTENANCE', '  just deps                     Install locked Go and frontend dependencies', '  just tidy                     Tidy Go modules and install frontend deps', '  just list                     Show this help')

[default]
[unix]
help:
    @printf '%s\n' 'LYN JUSTFILE(1)' '' 'NAME' '  just - show project command help' '' 'SYNOPSIS' '  just <command> [arguments]' '' 'QUALITY' '  just check                    Run all Go and frontend quality gates' '  just go-quality               Run gofmt check, tests, vet, and staticcheck' '  just frontend-quality         Run frontend formatting, lint, and type checks' '  just lint                     Run Go quality and frontend lint' '  just format                   Format Go and frontend files' '' 'BUILD AND RUN' '  just build                    Build the desktop app' '  just dev                      Start Wails development mode' '  just start                    Alias for dev' '  just scan                     Run the launcher from source' '  just install                  Install deps, build, and install the current working tree' '  just install-main             Sync clean main, install deps, build, and install' '' 'RELEASE' '  just bump [level]             Bump VERSION; level defaults to patch' '  just release [args...]        Package and publish a release' '  just release-patch [args...]  Bump patch, then release' '  just release-minor [args...]  Bump minor, then release' '  just release-major [args...]  Bump major, then release' '' 'MAINTENANCE' '  just deps                     Install locked Go and frontend dependencies' '  just tidy                     Tidy Go modules and install frontend deps' '  just list                     Show this help'

list: help

[windows]
check: go-quality-windows frontend-quality

[unix]
check: go-quality frontend-quality

[unix]
go-quality:
    files="$(find . -maxdepth 3 -name '*.go' -not -path './frontend/*')" && unformatted="$(gofmt -l $files)" && test -z "$unformatted" || { printf "%s\n" "$unformatted"; exit 1; }
    go test ./...
    go vet ./...
    staticcheck_bin="$(command -v staticcheck || true)"; test -n "$staticcheck_bin" || staticcheck_bin="$(go env GOPATH)/bin/staticcheck"; test -x "$staticcheck_bin" || { echo "==> Installing staticcheck" >&2; go install honnef.co/go/tools/cmd/staticcheck@latest; }; "$staticcheck_bin" ./...

[windows]
go-quality-windows:
    $goFiles = Get-ChildItem -LiteralPath . -Recurse -Filter *.go | Where-Object { $_.FullName -notmatch '\\node_modules\\' } | ForEach-Object { $_.FullName }; $files = gofmt -l $goFiles; if ($files) { $files; exit 1 }
    go test ./...
    go vet ./...
    $staticcheck = (Get-Command staticcheck -ErrorAction SilentlyContinue | Select-Object -First 1).Source; if (-not $staticcheck) { $staticcheckPath = Join-Path (go env GOPATH) "bin/staticcheck.exe"; if (Test-Path -LiteralPath $staticcheckPath) { $staticcheck = $staticcheckPath } }; if (-not $staticcheck) { Write-Output "==> Installing staticcheck"; go install honnef.co/go/tools/cmd/staticcheck@latest; $staticcheck = Join-Path (go env GOPATH) "bin/staticcheck.exe" }; & $staticcheck ./...

_playwright:
    @pnpm --dir {{ frontend }} exec playwright install chromium

frontend-quality: _playwright
    pnpm --dir {{ frontend }} quality

[windows]
_sync-main-windows:
    $branch = git branch --show-current; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; if ($branch -ne 'main') { throw "just install installs the dev version from main; current branch is $branch" }; $status = git status --porcelain; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; if ($status) { throw "just install installs the dev version from clean main; commit or stash local changes first" }; git fetch origin main; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }; git merge --ff-only origin/main; if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

[unix]
_sync-main-unix:
    branch="$(git branch --show-current)" && test "$branch" = main || { echo "just install installs the dev version from main; current branch is $branch" >&2; exit 1; }; test -z "$(git status --porcelain)" || { echo "just install installs the dev version from clean main; commit or stash local changes first" >&2; exit 1; }; git fetch origin main && git merge --ff-only origin/main

[unix]
format:
    find . -maxdepth 3 -name '*.go' -not -path './frontend/*' -exec gofmt -w {} +
    pnpm --dir {{ frontend }} format

[windows]
format:
    $goFiles = Get-ChildItem -LiteralPath . -Recurse -Filter *.go | Where-Object { $_.FullName -notmatch '\\node_modules\\' } | ForEach-Object { $_.FullName }; gofmt -w $goFiles
    pnpm --dir {{ frontend }} format

[unix]
lint: go-quality
    pnpm --dir {{ frontend }} lint

[windows]
lint: go-quality-windows
    pnpm --dir {{ frontend }} lint

_go-deps:
    @echo "==> Downloading Go modules"
    @go mod download

_frontend-deps:
    @echo "==> Installing frontend dependencies"
    @pnpm --dir {{ frontend }} install --frozen-lockfile --prefer-offline

deps: _go-deps _frontend-deps

_frontend-build: _frontend-deps
    @echo "==> Building frontend assets"
    @pnpm --dir {{ frontend }} build

[windows]
build: _frontend-build
    @echo "==> Building Lyn for Windows"
    @New-Item -ItemType Directory -Path "{{ build-bin }}" -Force | Out-Null
    @$wails = Get-Command wails -ErrorAction SilentlyContinue; if ($wails) { wails build -m -nosyncgomod -s } else { go build -tags "desktop,production" -ldflags="-w -s -H windowsgui" -o "{{ build-bin }}/lyn.exe" . }

[linux]
build: _frontend-build
    @echo "==> Building Lyn"
    @bash "{{ justfile_directory() }}/scripts/build-linux.sh" "{{ justfile_directory() }}" "{{ build-bin }}"

[macos]
build: _frontend-build
    @echo "==> Building Lyn"
    @mkdir -p {{ build-bin }}
    @if command -v wails >/dev/null 2>&1; then wails build -m -nosyncgomod -s; else go build -tags "desktop,production" -o {{ build-bin }}/lyn .; fi

[windows]
install: build
    @echo "==> Installing Lyn"
    @& (Join-Path "{{ justfile_directory() }}" "scripts/install-windows.ps1") -Root "{{ justfile_directory() }}"

[windows]
install-main: _sync-main-windows build
    @echo "==> Installing Lyn"
    @& (Join-Path "{{ justfile_directory() }}" "scripts/install-windows.ps1") -Root "{{ justfile_directory() }}"

[linux]
install: build
    @echo "==> Installing Lyn"
    @bash "{{ justfile_directory() }}/scripts/install-linux.sh" "{{ justfile_directory() }}"

[linux]
install-main: _sync-main-unix build
    @echo "==> Installing Lyn"
    @bash "{{ justfile_directory() }}/scripts/install-linux.sh" "{{ justfile_directory() }}"

[macos]
install:
    @echo "unsupported install target macos" >&2; exit 1

[windows]
dev:
    if (Get-Command wails -ErrorAction SilentlyContinue) { wails dev } else { & (Join-Path (go env GOPATH) "bin/wails") dev }

[unix]
dev:
    if command -v wails >/dev/null 2>&1; then wails dev; else "$(go env GOPATH)/bin/wails" dev; fi

start: dev

scan:
    go run .

[unix]
bump level="patch":
    bash "{{ justfile_directory() }}/scripts/release.sh" bump "{{ level }}"

[windows]
bump level="patch":
    & (Join-Path "{{ justfile_directory() }}" "scripts/release-windows.ps1") bump "{{ level }}"

[linux]
package-deb: build
    bash "{{ justfile_directory() }}/scripts/release.sh" package-deb

[windows]
package-deb: _frontend-build
    & (Join-Path "{{ justfile_directory() }}" "scripts/release-windows.ps1") package-deb

[windows]
package-windows: build
    & (Join-Path "{{ justfile_directory() }}" "scripts/release-windows.ps1") package-windows

[windows]
dev-sign-setup:
    @& (Join-Path "{{ justfile_directory() }}" "scripts/dev-sign.ps1") -Command setup

[windows]
dev-sign: build
    @& (Join-Path "{{ justfile_directory() }}" "scripts/dev-sign.ps1") -Command uiaccess -Path "{{ build-bin }}/lyn.exe" -Manifest "build/windows/wails.exe.uiaccess.manifest"

[linux]
test-deb-install deb="":
    bash "{{ justfile_directory() }}/scripts/release.sh" test-deb-install "{{ deb }}"

[linux]
publish-assets tag="":
    bash "{{ justfile_directory() }}/scripts/release.sh" publish-assets "{{ tag }}"

[windows]
publish-assets tag="":
    & (Join-Path "{{ justfile_directory() }}" "scripts/release-windows.ps1") publish-assets "{{ tag }}"

[unix]
release *args:
    bash "{{ justfile_directory() }}/scripts/release.sh" release {{ args }}

[windows]
release *args:
    & (Join-Path "{{ justfile_directory() }}" "scripts/release-windows.ps1") release {{ args }}

[unix]
release-patch *args:
    bash "{{ justfile_directory() }}/scripts/release.sh" release --bump patch {{ args }}

[windows]
release-patch *args:
    & (Join-Path "{{ justfile_directory() }}" "scripts/release-windows.ps1") release --bump patch {{ args }}

[unix]
release-minor *args:
    bash "{{ justfile_directory() }}/scripts/release.sh" release --bump minor {{ args }}

[windows]
release-minor *args:
    & (Join-Path "{{ justfile_directory() }}" "scripts/release-windows.ps1") release --bump minor {{ args }}

[unix]
release-major *args:
    bash "{{ justfile_directory() }}/scripts/release.sh" release --bump major {{ args }}

[windows]
release-major *args:
    & (Join-Path "{{ justfile_directory() }}" "scripts/release-windows.ps1") release --bump major {{ args }}

tidy:
    go mod tidy
    pnpm --dir {{ frontend }} install
