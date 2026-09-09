#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-project-contract-v2.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; export GOCACHE

(cd "$repo/reference/go" && go test -count=1 -buildvcs=false ./projectmodule && go run -buildvcs=false ./cmd/project-module -version 2 -out "$work/module.g1")
cmp "$repo/modules/project/v2/module.g1" "$work/module.g1"
(cd "$repo/modules/project/v2" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/project/v2/module.seme" "$work/module.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/module.seme"
"$repo/scripts/check-semantic-module-registry.sh"

echo 'Project Contract v2: deterministic ancestry, artifacts, hashes, Foundation validation, and collision audit passed'
