#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-project-v6.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
(cd "$repo/reference/go" && GOCACHE="$work/go-cache" go test -count=1 ./projectmodule ./contractcatalog && GOCACHE="$work/go-cache" go run ./cmd/project-module -version 6 > "$work/module.g1")
cmp "$repo/modules/project/v6/module.g1" "$work/module.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/project/v6/module.seme" "$work/module.seme"
(cd "$repo/modules/project/v6" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$repo/modules/project/v6/module.seme"
"$repo/scripts/check-semantic-module-registry.sh"
node -e 'const r=require(process.argv[1]),f=r.families.find(x=>x.family==="project"),v=f&&f.revisions.find(x=>x.directory==="v6");if(!v||v.revision!=="0000000000000000000000000000e008"||!v.imports.some(x=>x.module==="00000000000000000000000000004000"&&x.revision==="00000000000000000000000000004001"))process.exit(1)' "$repo/conformance/semantic-module-registry-v1.json"
echo 'Project Contract v6: immutable ancestry, Configuration v1 binding, canonical validation, and registry ownership pass'
