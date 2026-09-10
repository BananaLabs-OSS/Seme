#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-durable-state-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
(cd "$repo/reference/go" && GOCACHE="$work/cache" go test -count=1 ./durablestatemodule ./contractcatalog && GOCACHE="$work/cache" go run ./cmd/durable-state-module > "$work/module.g1")
cmp "$repo/modules/durable-state/v1/module.g1" "$work/module.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/durable-state/v1/module.seme" "$work/module.seme"
(cd "$repo/modules/durable-state/v1" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$repo/modules/durable-state/v1/module.seme"
echo 'Durable State Contract v1: neutral versioned state and storage port contract is reproducible'
