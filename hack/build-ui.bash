#!/usr/bin/env bash

set -e

DEVSPACE_ROOT=$(git rev-parse --show-toplevel)

# Install dependencies
cd ui && npm install && npm run build

# Pack ui
echo "Packing ui"
mkdir -p "${DEVSPACE_ROOT}/release"
tar -C "${DEVSPACE_ROOT}/ui/build" -czf "${DEVSPACE_ROOT}/release/ui.tar.gz" .
shasum -a 256 "${DEVSPACE_ROOT}/release/ui.tar.gz" > "${DEVSPACE_ROOT}/release/ui.tar.gz".sha256
