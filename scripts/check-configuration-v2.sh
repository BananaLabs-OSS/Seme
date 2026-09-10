#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-configuration-v2.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
(cd "$repo/reference/go" && GOCACHE="$work/go-cache" go test -count=1 ./configurationmodule ./contractcatalog && GOCACHE="$work/go-cache" go run ./cmd/configuration-module -version 2 > "$work/module.g1")
cmp "$repo/modules/configuration/v2/module.g1" "$work/module.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/configuration/v2/module.seme" "$work/module.seme"
(cd "$repo/modules/configuration/v2" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$repo/modules/configuration/v2/module.seme"
"$repo/scripts/check-semantic-module-registry.sh"
echo 'Configuration Contract v2: immutable ancestry, typed initializer bindings, canonical validation, and registry ownership pass'
