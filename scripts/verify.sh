#!/usr/bin/env bash
# Build, unit tests and coverage gate for oficina-api.
set -euo pipefail

THRESHOLD="${COVERAGE_THRESHOLD:-90}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PACKAGES=(
  "./internal/services/..."
  "./internal/handlers/..."
  "./internal/repository/..."
)

echo "▶ go build"
go build ./...

echo "▶ go test (escopo de cobertura)"
COVER_FILE="$(mktemp -t oficina-cover.XXXXXX.out)"
go test "${PACKAGES[@]}" -coverprofile="$COVER_FILE" -covermode=atomic

TOTAL_LINE="$(go tool cover -func="$COVER_FILE" | awk '/^total:/ {print $3}' | tr -d '%')"
rm -f "$COVER_FILE"

echo "▶ cobertura agregada: ${TOTAL_LINE}% (mínimo ${THRESHOLD}%)"

awk -v total="$TOTAL_LINE" -v min="$THRESHOLD" 'BEGIN {
  if (total + 0 < min + 0) {
    printf "❌ Cobertura %.1f%% abaixo de %d%%. Veja docs/QUALIDADE.md\n", total, min
    exit 1
  }
  printf "✅ Build, testes e cobertura OK (%.1f%%)\n", total
}'
