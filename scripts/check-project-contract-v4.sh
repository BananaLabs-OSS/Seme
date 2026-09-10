#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd);work=$(mktemp -d "${TMPDIR:-/tmp}/seme-project-v4.XXXXXX");trap 'rm -rf "$work"' EXIT HUP INT TERM
(cd "$repo/reference/go" && GOCACHE="$work/cache" go test -count=1 ./projectmodule && GOCACHE="$work/cache" go run ./cmd/project-module --version 4 > "$work/module.g1")
cmp "$repo/modules/project/v4/module.g1" "$work/module.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme";cmp "$repo/modules/project/v4/module.seme" "$work/module.seme"
(cd "$repo/modules/project/v4" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
# Like earlier Project revisions, this contract contains schema constraints to
# exact imported modules and is validated compositionally by Foundation plus
# the semantic-module registry rather than as a closed Kernel-only envelope.
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$repo/modules/project/v4/module.seme"
"$repo/scripts/check-semantic-module-registry.sh"
node -e 'const r=require(process.argv[1]),f=r.families.find(x=>x.family==="project"),v=f&&f.revisions.find(x=>x.revision.endsWith("e004"));if(!v||v.parents.length!==1||!v.parents[0].endsWith("e003")||!v.imports.some(x=>x.module.endsWith("f000")&&x.revision.endsWith("f001")))process.exit(1)' "$repo/conformance/semantic-module-registry-v1.json"
echo 'Project Contract v4: additive dependency-closure binding and exact imports pass'
