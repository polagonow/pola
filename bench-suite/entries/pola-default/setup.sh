#!/usr/bin/env bash
# Install the Pola app's frontend dependencies (Go modules are resolved by
# `go build` during the build step). Idempotent.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
cd "$here/app/web"
pnpm install
