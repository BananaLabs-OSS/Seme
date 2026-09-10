#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd); work=$(mktemp -d "${TMPDIR:-/tmp}/seme-project-v5.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
(cd "$repo/reference/go" && GOCACHE="$work/go" go run ./cmd/project-module --version 5 --out "$work/v5.g1")
cmp "$repo/modules/project/v5/module.g1" "$work/v5.g1"; "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/v5.g1" "$work/v5.seme"; cmp "$repo/modules/project/v5/module.seme" "$work/v5.seme"
(cd "$repo/modules/project/v5" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$repo/modules/project/v5/module.seme"
(cd "$repo/reference/go" && GOCACHE="$work/go" go test ./projectmodule ./contractcatalog)
"$repo/scripts/check-semantic-module-registry.sh"
node -e 'const r=require(process.argv[1]),f=r.families.find(x=>x.family==="project"),v=f&&f.revisions.find(x=>x.revision.endsWith("e005"));if(!v||v.parents.length!==1||!v.parents[0].endsWith("e004")||!v.imports.some(x=>x.module.endsWith("b000")&&x.revision.endsWith("b003")))process.exit(1)' "$repo/conformance/semantic-module-registry-v1.json"
echo 'Project Contract v5: authenticated complete package graph binding passes'
