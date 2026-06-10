#!/usr/bin/env bash
set -euo pipefail

OLH_BIN="${OLH_BIN:-olh}"
PROVIDER_ID="${PROVIDER_ID:-simulated-tag-1}"
PROVIDER_TYPE="${PROVIDER_TYPE:-virtual}"
SOURCE_ID="${SOURCE_ID:-simulated-tag-1}"
SLEEP_SECONDS="${SLEEP_SECONDS:-3}"

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT

payloads=(
  '{"position":{"type":"Point","coordinates":[8.903330377938914,52.01757421663987]},"crs":"EPSG:4326","provider_id":"__PROVIDER_ID__","provider_type":"__PROVIDER_TYPE__","source":"__SOURCE_ID__"}'
  '{"position":{"type":"Point","coordinates":[8.90318,52.01776]},"crs":"EPSG:4326","provider_id":"__PROVIDER_ID__","provider_type":"__PROVIDER_TYPE__","source":"__SOURCE_ID__"}'
  '{"position":{"type":"Point","coordinates":[8.903065,52.01779]},"crs":"EPSG:4326","provider_id":"__PROVIDER_ID__","provider_type":"__PROVIDER_TYPE__","source":"__SOURCE_ID__"}'
  '{"position":{"type":"Point","coordinates":[8.90322,52.01788]},"crs":"EPSG:4326","provider_id":"__PROVIDER_ID__","provider_type":"__PROVIDER_TYPE__","source":"__SOURCE_ID__"}'
)

for i in "${!payloads[@]}"; do
  payload="${payloads[$i]}"
  payload="${payload//__PROVIDER_ID__/$PROVIDER_ID}"
  payload="${payload//__PROVIDER_TYPE__/$PROVIDER_TYPE}"
  payload="${payload//__SOURCE_ID__/$SOURCE_ID}"

  file="$tmpdir/update-$((i + 1)).json"
  printf '[%s]\n' "$payload" > "$file"

  echo "posting update $((i + 1)) via $OLH_BIN"
  "$OLH_BIN" locations post --json -f "$file"

  if [[ "$i" -lt $((${#payloads[@]} - 1)) ]]; then
    sleep "$SLEEP_SECONDS"
  fi
done
