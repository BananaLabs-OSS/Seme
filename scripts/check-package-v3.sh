#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd); work=$(mktemp -d "${TMPDIR:-/tmp}/seme-package-v3.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
(cd "$repo/reference/go" && GOCACHE="$work/go" go run ./cmd/package-module --version 1 > "$work/v1.g1" && GOCACHE="$work/go" go run ./cmd/package-module --version 2 > "$work/v2.g1" && GOCACHE="$work/go" go run ./cmd/package-module --version 3 > "$work/v3.g1")
cmp "$repo/modules/package/v1/module.g1" "$work/v1.g1"; cmp "$repo/modules/package/v2/module.g1" "$work/v2.g1"; cmp "$repo/modules/package/v3/module.g1" "$work/v3.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/v3.g1" "$work/v3.seme"; cmp "$repo/modules/package/v3/module.seme" "$work/v3.seme"
(cd "$repo/modules/package/v3" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$repo/modules/package/v3/module.seme"; "$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$repo/modules/package/v3/module.seme"
(cd "$repo/reference/go" && GOCACHE="$work/go" go test ./contractcatalog ./cmd/package-module)
echo 'Package Contract v3: additive declaration ownership and reproduction pass'
