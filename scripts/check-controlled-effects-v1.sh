#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-controlled-effects-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
(cd "$repo/reference/go" && GOCACHE="$work/cache" go test -count=1 ./controlledeffectsmodule ./contractcatalog && GOCACHE="$work/cache" go run ./cmd/controlled-effects-module > "$work/module.g1")
cmp "$repo/modules/controlled-effects/v1/module.g1" "$work/module.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/controlled-effects/v1/module.seme" "$work/module.seme"
(cd "$repo/modules/controlled-effects/v1" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$repo/modules/controlled-effects/v1/module.seme"
node "$repo/scripts/semantic-module-registry.mjs" --root "$repo/modules" --out "$work/registry.json"
echo 'Controlled Effects Contract v1: injected clock, seeded randomness, external Boolean effect, replay, and bounds are reproducible'
