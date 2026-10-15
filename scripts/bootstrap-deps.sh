#!/usr/bin/env bash
# Download the published dependencies for the library and both example modules.
set -euo pipefail
cd "$(dirname "$0")/.."
for module in . examples/gallery examples/transport; do
  go -C "$module" mod download
done
