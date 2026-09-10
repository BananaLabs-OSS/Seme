#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-project-v11.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
(cd "$repo/reference/go" && GOCACHE="$work/cache" go test -count=1 ./projectmodule ./contractcatalog && GOCACHE="$work/cache" go run ./cmd/project-module -version 11 > "$work/module.g1")
cmp "$repo/modules/project/v11/module.g1" "$work/module.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/project/v11/module.seme" "$work/module.seme"
(cd "$repo/modules/project/v11" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$repo/modules/project/v11/module.seme"
echo 'Project Contract v11: exact Project-v10 and neutral Ordered Transport-v1 binding is reproducible'
