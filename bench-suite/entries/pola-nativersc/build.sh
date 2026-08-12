#!/usr/bin/env bash
# Build the Pola app into a single static binary at ./server-bin.
# Regenerates the JS bridge first (picks up new Go actions), then runs the
# two-stage `pola build` (bundle assets → compile Go binary with embedded assets).
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
pola="$here/../../../bin/pola"    # repo-root bin/pola (built via `mage build`)

if [[ ! -x "$pola" ]]; then
  echo "pola CLI not found at $pola — run 'go run mage.go build' at the repo root first." >&2
  exit 1
fi

cd "$here/app"

# Pick up actions/*.go changes in the TS bridge.
"$pola" generate js:bridge

# Production build → single binary. goja needs no CGO (fully static).
CGO_ENABLED=0 "$pola" build \
  --vm goja --renderer nativersc --bundler esbuild --router nextjs --css none \
  --cgo 0 \
  -o "$here/server-bin"

echo "built $here/server-bin"
