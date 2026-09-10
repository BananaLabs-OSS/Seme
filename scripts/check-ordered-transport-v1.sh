#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-ordered-transport-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
(cd "$repo/reference/go" && GOCACHE="$work/cache" go test -count=1 ./orderedtransportmodule ./orderedtransportinstance ./contractcatalog && GOCACHE="$work/cache" go run ./cmd/ordered-transport-module > "$work/module.g1")
cmp "$repo/modules/ordered-transport/v1/module.g1" "$work/module.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/ordered-transport/v1/module.seme" "$work/module.seme"
(cd "$repo/modules/ordered-transport/v1" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$repo/modules/ordered-transport/v1/module.seme"
if rg -qi 'http|websocket|socket|sse|messagepack|tcp|udp|filesystem|database|javascript|golang' "$repo/modules/ordered-transport/v1/module.g1"; then
	echo 'Ordered Transport contract contains mechanism-specific vocabulary' >&2
	exit 1
fi
echo 'Ordered Transport Contract v1: neutral bounded sequencing, correlation, replay, and whole-frame port authority is reproducible'
