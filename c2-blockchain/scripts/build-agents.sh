#!/usr/bin/env bash
# Cross-compile C2 agent for lab targets (MVP-07 multi-OS).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/dist"
mkdir -p "$OUT"

echo "Building agents into $OUT"
GOOS=linux GOARCH=amd64 go build -o "$OUT/c2-agent-linux-amd64" "$ROOT/cmd/agent"
GOOS=linux GOARCH=arm64 go build -o "$OUT/c2-agent-linux-arm64" "$ROOT/cmd/agent"
GOOS=windows GOARCH=amd64 go build -o "$OUT/c2-agent-windows-amd64.exe" "$ROOT/cmd/agent"

echo "Done:"
ls -la "$OUT"
