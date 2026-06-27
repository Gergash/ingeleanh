#!/usr/bin/env bash
# Quick Tunnel para portal C2 (sin dominio). Requiere: go run ./cmd/server en :8443
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONFIG="$ROOT/cloudflared/config.quick.yml"
ORIGIN="${C2_TUNNEL_ORIGIN:-http://localhost:8443}"

if [[ -f "/c/Program Files (x86)/cloudflared/cloudflared.exe" ]]; then
	CF="/c/Program Files (x86)/cloudflared/cloudflared.exe"
elif command -v cloudflared >/dev/null 2>&1; then
	CF=cloudflared
else
	echo "cloudflared no encontrado. Instala: winget install Cloudflare.cloudflared" >&2
	exit 1
fi

EXTRA=()
CA_BUNDLE="/c/Program Files/Git/usr/ssl/certs/ca-bundle.crt"
if [[ -f "$CA_BUNDLE" ]]; then
	EXTRA+=(--origin-ca-pool "$CA_BUNDLE")
fi

echo "Origen local: $ORIGIN"
echo "Config: $CONFIG"
echo "Espera la URL https://....trycloudflare.com y abre /portal/"
echo ""

exec "$CF" tunnel \
	--config "$CONFIG" \
	--url "$ORIGIN" \
	--no-prechecks \
	--loglevel info \
	--transport-loglevel error \
	--management-diagnostics=false \
	--metrics "127.0.0.1:0" \
	"${EXTRA[@]}"
