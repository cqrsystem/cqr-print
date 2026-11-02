#!/usr/bin/env bash
# Compila o cqrprint.exe para Windows (de qualquer sistema com Go 1.24+).
# Uso: scripts/compilar.sh [versao] [servidor padrão]
set -euo pipefail
cd "$(dirname "$0")/.."
VERSAO="${1:-0.0.0-dev}"
SERVIDOR="${2:-}"
windows/recursos.sh "$VERSAO"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -ldflags "-X main.versao=${VERSAO} -X main.servidorPadrao=${SERVIDOR}" \
  -o dist/cqrprint.exe ./cmd/cqrprint
echo "dist/cqrprint.exe ($VERSAO)"
