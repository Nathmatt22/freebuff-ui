#!/usr/bin/env bash
# Build the self-contained injector exe.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
mkdir -p "$here/dist"

echo "==> checking theme engine syntax"
if command -v node >/dev/null 2>&1; then
  node --check "$here/injector/assets/theme-engine.js"
  echo "    ok"
else
  echo "    (node not found, skipping)"
fi

echo "==> building injector"
cd "$here/injector"
go build -trimpath -ldflags "-s -w" -o "$here/dist/FreebuffThemeInjector.exe" .

echo "==> done"
ls -la "$here/dist"
