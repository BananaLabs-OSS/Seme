#!/bin/sh
# Bind two independently authenticated UPB12 project authorities and the shared
# three-language native validation into one neutral Project-v14 revision.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
prior=${1:?prior build export required}; result=${2:?result build export required}; reconciliation=${3:?shared reconciliation required}; destination=${4:?new Project-v14 destination required}
test ! -e "$destination"; work=$(mktemp -d "${TMPDIR:-/tmp}/seme-upb12-v14.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; export GOCACHE
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/finalize" ./cmd/project-v14-finalize)
finalize(){ out=$1; "$work/finalize" \
  -bundle "$prior/result-base" -placement "$prior/result-placement" \
  -selection "$prior/inputs/configuration-selection.json" -durable-selection "$prior/inputs/durable-selection.json" -transport-selection "$prior/inputs/transport-selection.json" -effects-selection "$prior/inputs/controlled-effects-selection.json" \
  -result-bundle "$result/result-base" -result-placement "$result/result-placement" -result-selection "$result/inputs/configuration-selection.json" \
  -reconciliation "$reconciliation" -language shared -project-identity seme.upb12/service -target-name wasm32-pulp-upb12-host-v1 -rule-namespace upb12 -reconciliation-label seme-upb12-shared-reconciliation-v1 -identity-evidence-required=false \
  -foundation "$repo/modules/foundation/v1/module.seme" -execution "$repo/modules/execution/v36/module.seme" -package "$repo/modules/package/v4/module.seme" -dependency "$repo/modules/dependency/v1/module.seme" -configuration "$repo/modules/configuration/v3/module.seme" -resource "$repo/modules/resource/v1/module.seme" -durable-state "$repo/modules/durable-state/v1/module.seme" -source-presentation "$repo/modules/source-presentation/v1/module.seme" -ordered-transport "$repo/modules/ordered-transport/v1/module.seme" -controlled-effects "$repo/modules/controlled-effects/v1/module.seme" \
  -target "$repo/modules/target/v1/module.seme" -project-v8 "$repo/modules/project/v8/module.seme" -project-v9 "$repo/modules/project/v9/module.seme" -project-v10 "$repo/modules/project/v10/module.seme" -project-v11 "$repo/modules/project/v11/module.seme" -project-v12 "$repo/modules/project/v12/module.seme" -project-v13 "$repo/modules/project/v13/module.seme" -patch "$repo/modules/patch/v1/module.seme" -language-service "$repo/modules/language-service/v1/module.seme" -project-v14 "$repo/modules/project/v14/module.seme" \
  -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0" -out "$out"; }
finalize "$work/final-a"; finalize "$work/final-b"; diff -ru "$work/final-a" "$work/final-b"
if finalize "$work/final-a" > "$work/collision.out" 2> "$work/collision.err"; then echo 'UPB12 Project-v14 overwrote an existing authority' >&2; exit 1; fi
test ! -s "$work/collision.out"; test -s "$work/final-a/patch-v1.seme"; test -s "$work/final-a/project-v14.seme"
parent=$(dirname "$destination"); test "$(cd "$parent" && pwd -P)" = "$parent"; mv -T -n "$work/final-a" "$destination"; test ! -e "$work/final-a"
printf 'UPB12 Project-v14: one neutral patch binds the shared native validation and exact before/after authorities\n'
