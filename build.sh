#!/usr/bin/env bash
# Build the self-contained injector for the host platform.
#
#   ./build.sh                 native binary (Linux/macOS/Windows)
#   ./build.sh --linux         force a Linux amd64 build
#   ./build.sh --linux-arm64   force a Linux arm64 build
#   ./build.sh --darwin        force a macOS build
#   ./build.sh --windows       force a Windows build
#   ./build.sh --all           every target into dist/
#
# Cross-compiling needs no toolchain beyond a Go install: the injector uses
# only the standard library, and the platform-specific code is selected with
# build tags.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
mkdir -p "$here/dist"

go_bin="${GO:-go}"
if ! command -v "$go_bin" >/dev/null 2>&1; then
  echo "==> go not found in PATH. Install Go 1.21+ or set GO=/path/to/go." >&2
  exit 1
fi

echo "==> checking theme engine syntax"
node --check "$here/injector/assets/theme-engine.js"
echo "    ok"

# build <goos> <goarch> <outfile>
build() {
  local goos="$1" goarch="$2" out="$3"
  echo "==> building $goos/$goarch -> dist/$out"
  (cd "$here/injector" && GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
    "$go_bin" build -trimpath -ldflags "-s -w" -o "$here/dist/$out" .)
}

target="${1:-native}"
case "$target" in
  native)
    goos="$("$go_bin" env GOOS)"
    goarch="$("$go_bin" env GOARCH)"
    case "$goos" in
      windows) ext=".exe" ;;
      *) ext="" ;;
    esac
    build "$goos" "$goarch" "FreebuffThemeInjector$ext"
    ;;
  --linux|--linux-amd64)
    build linux amd64 FreebuffThemeInjector
    ;;
  --linux-arm64)
    build linux arm64 FreebuffThemeInjector-arm64
    ;;
  --darwin|--darwin-arm64)
    build darwin arm64 FreebuffThemeInjector-darwin
    ;;
  --windows)
    build windows amd64 FreebuffThemeInjector.exe
    ;;
  --all)
    build linux amd64 FreebuffThemeInjector
    build linux arm64 FreebuffThemeInjector-arm64
    build darwin arm64 FreebuffThemeInjector-darwin
    build darwin amd64 FreebuffThemeInjector-darwin-amd64
    build windows amd64 FreebuffThemeInjector.exe
    ;;
  *)
    echo "unknown option: $target (try --linux, --darwin, --windows, --all)" >&2
    exit 2
    ;;
esac

echo "==> done"
ls -la "$here/dist"