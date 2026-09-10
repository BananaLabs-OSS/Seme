#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-configuration-v3.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
(cd "$repo/reference/go" && GOCACHE="$work/cache" go test -count=1 ./configurationmodule ./contractcatalog)
(cd "$repo/reference/go" && GOCACHE="$work/cache" go run ./cmd/configuration-module -version 3) > "$work/module.g1"
cmp "$repo/modules/configuration/v3/module.g1" "$work/module.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/configuration/v3/module.seme" "$work/module.seme"
(cd "$repo/modules/configuration/v3" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$repo/modules/configuration/v3/module.seme"
echo 'Configuration Contract v3: Package v4 and Execution v36 pins are reproducible'
