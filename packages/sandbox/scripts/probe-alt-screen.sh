#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

echo "Running alternate screen probe..."
cargo run --quiet --bin alt_screen_probe
