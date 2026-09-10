#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-dependency-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
(cd "$repo/reference/go" && GOCACHE="$work/go-cache" go test -count=1 ./dependencymodule && GOCACHE="$work/go-cache" go run ./cmd/dependency-module > "$work/module.g1")
cmp "$repo/modules/dependency/v1/module.g1" "$work/module.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/dependency/v1/module.seme" "$work/module.seme"
(cd "$repo/modules/dependency/v1" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$repo/modules/dependency/v1/module.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$repo/modules/dependency/v1/module.seme"
"$repo/scripts/check-semantic-module-registry.sh"
node -e 'const r=require(process.argv[1]),f=r.families.find(x=>x.family==="dependency");if(!f||f.module!=="0000000000000000000000000000f000"||f.revisions.length!==1||f.revisions[0].revision!=="0000000000000000000000000000f001")process.exit(1)' "$repo/conformance/semantic-module-registry-v1.json"
echo 'Dependency Contract v1: deterministic generation, canonical validation, closed kind enum, and registry ownership pass'
