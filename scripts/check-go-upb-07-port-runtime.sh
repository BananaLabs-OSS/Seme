#!/bin/sh
# Host-boundary evidence only. This does not claim a Pulp DurablePort provider.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb07-port.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; export GOCACHE
(cd "$repo/reference/go" && go test -race -count=1 ./durableruntime)
(cd "$repo/reference/go" && go test -count=10 ./durableruntime)
if rg -n '(^|[^[:alnum:]_])(os|time|rand)\.|database/sql|sqlite|go[[:space:]]+func' "$repo/reference/go/durableruntime" --glob '*.go' | rg -v '_test\.go:'; then
  echo 'durable runtime boundary gained ambient/provider-specific behavior' >&2
  exit 1
fi
echo 'Go UPB-07 DurablePort boundary: authenticated profile, bounded canonical payloads, opaque-token Load/CAS, exact traces, atomic memory adapter, and adversaries pass; pinned Pulp has no matching CAS realization and is not claimed'
