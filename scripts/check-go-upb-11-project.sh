#!/bin/sh
# Full real-project UPB11 reconciliation, rebuild, placement, and v14 proof.
# This is an additive partition; only check-go-upb-11.sh may claim the cell.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb11-project.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"
export GOCACHE XDG_CACHE_HOME

"$repo/scripts/materialize-go-upb09-fixture.sh" "$work/source"
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/build09" ./cmd/go-upb09-build)
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/place" ./cmd/go-upb10-place)
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/reconcile" ./cmd/go-upb11-reconcile)
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/finalize" ./cmd/go-upb11-finalize)
(cd "$repo/reference/go" && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -buildvcs=false -o "$work/canonical-vm.wasm" ./cmd/canonical-wasm-cell)
cp "$repo/targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml" "$work/pulp.cell.toml"

build09() {
  project=$1; revision=$2; destination=$3; bindings=${4:-}
  if test -n "$bindings"; then identity="-identity-bindings=$bindings"; else identity=; fi
  "$work/build09" \
    -project "$project" -proxy "$repo/fixtures/go-upb03-offline-proxy" \
    -module example.test/go-uab-11 \
    -package example.test/go-uab-11/streamservice -entry DispatchControlled \
    -revision "$revision" -out "$destination" $identity \
    -dependency example.test/seme/checksum -version v1.2.3 \
    -local-from example.test/go-uab-11/application -local-to example.test/go-uab-11/model \
    -selection "$project/configuration-selection.json" \
    -resources "$project/resources.json" -resource-owner example.test/go-uab-11/service \
    -durable-selection "$project/durable-selection.json" \
    -transport-selection "$repo/fixtures/go-upb08-transport-overlay/transport-selection.json" \
    -effects-selection "$project/controlled-effects-selection.json" \
    -execution-g1 "$repo/modules/execution/v36/module.g1" \
    -execution-contract "$repo/modules/execution/v36/module.seme" \
    -foundation-contract "$repo/modules/foundation/v1/module.seme" \
    -package-v4 "$repo/modules/package/v4/module.seme" \
    -dependency-v1 "$repo/modules/dependency/v1/module.seme" \
    -configuration-v3 "$repo/modules/configuration/v3/module.seme" \
    -resource-v1 "$repo/modules/resource/v1/module.seme" \
    -durable-state-v1 "$repo/modules/durable-state/v1/module.seme" \
    -source-presentation-v1 "$repo/modules/source-presentation/v1/module.seme" \
    -ordered-transport-v1 "$repo/modules/ordered-transport/v1/module.seme" \
    -controlled-effects-v1 "$repo/modules/controlled-effects/v1/module.seme" \
    -project-v8 "$repo/modules/project/v8/module.seme" \
    -project-v9 "$repo/modules/project/v9/module.seme" \
    -project-v10 "$repo/modules/project/v10/module.seme" \
    -project-v11 "$repo/modules/project/v11/module.seme" \
    -project-v12 "$repo/modules/project/v12/module.seme" \
    -k0 "$repo/bootstrap/seme-k0-linux-amd64" \
    -g1-compiler "$repo/compiler/g1-compiler.k0"
}

printf 'UPB-11 project stage: base build and semantic reconciliation\n'
build09 "$work/source" 1 "$work/prior-base"
"$work/reconcile" \
  -project "$work/source" -out "$work/result-source" \
  -module example.test/go-uab-11 \
  -package example.test/go-uab-11/streamservice -entry DispatchControlled -revision 2 \
  -execution-g1 "$repo/modules/execution/v36/module.g1" \
  -provider-g1 "$repo/modules/provider/v1/module.g1" \
  -target example.test/go-uab-11/transport.EqualI64 \
  -expected EqualI64 -replacement SameI64
build09 "$work/result-source" 2 "$work/result-base" "$work/result-source/.seme-reconciliation-v1/projection-report.json"

common_place() {
  bundle=$1; shift
  "$work/place" -bundle "$bundle" \
    -selection "$work/source/configuration-selection.json" \
    -durable-selection "$work/source/durable-selection.json" \
    -transport-selection "$repo/fixtures/go-upb08-transport-overlay/transport-selection.json" \
    -effects-selection "$work/source/controlled-effects-selection.json" \
    -foundation "$repo/modules/foundation/v1/module.seme" \
    -execution "$repo/modules/execution/v36/module.seme" \
    -package "$repo/modules/package/v4/module.seme" \
    -dependency "$repo/modules/dependency/v1/module.seme" \
    -configuration "$repo/modules/configuration/v3/module.seme" \
    -resource "$repo/modules/resource/v1/module.seme" \
    -durable-state "$repo/modules/durable-state/v1/module.seme" \
    -source-presentation "$repo/modules/source-presentation/v1/module.seme" \
    -ordered-transport "$repo/modules/ordered-transport/v1/module.seme" \
    -controlled-effects "$repo/modules/controlled-effects/v1/module.seme" \
    -target "$repo/modules/target/v1/module.seme" \
    -project-v8 "$repo/modules/project/v8/module.seme" \
    -project-v9 "$repo/modules/project/v9/module.seme" \
    -project-v10 "$repo/modules/project/v10/module.seme" \
    -project-v11 "$repo/modules/project/v11/module.seme" \
    -project-v12 "$repo/modules/project/v12/module.seme" \
    -project-v13 "$repo/modules/project/v13/module.seme" \
    -k0 "$repo/bootstrap/seme-k0-linux-amd64" \
    -g1-compiler "$repo/compiler/g1-compiler.k0" \
    -canonical-vm "$work/canonical-vm.wasm" -pulp-cell "$work/pulp.cell.toml" "$@"
}
printf 'UPB-11 project stage: independently place both project revisions\n'
common_place "$work/prior-base" -out "$work/prior-placement"
common_place "$work/result-base" -out "$work/result-placement"

finalize() {
  destination=$1
  "$work/finalize" \
    -prior-bundle "$work/prior-base" -prior-placement "$work/prior-placement" \
    -result-bundle "$work/result-base" -result-placement "$work/result-placement" \
    -reconciliation "$work/result-source" \
    -module example.test/go-uab-11 -root-package example.test/go-uab-11/streamservice \
    -entry DispatchControlled -client-revision 2 \
    -provider-g1 "$repo/modules/provider/v1/module.g1" \
    -execution-g1 "$repo/modules/execution/v36/module.g1" \
    -patch "$repo/modules/patch/v1/module.seme" \
    -language-service "$repo/modules/language-service/v1/module.seme" \
    -project-v14 "$repo/modules/project/v14/module.seme" -out "$destination" \
    -selection "$work/source/configuration-selection.json" \
    -durable-selection "$work/source/durable-selection.json" \
    -transport-selection "$repo/fixtures/go-upb08-transport-overlay/transport-selection.json" \
    -effects-selection "$work/source/controlled-effects-selection.json" \
    -foundation "$repo/modules/foundation/v1/module.seme" \
    -execution "$repo/modules/execution/v36/module.seme" \
    -package "$repo/modules/package/v4/module.seme" \
    -dependency "$repo/modules/dependency/v1/module.seme" \
    -configuration "$repo/modules/configuration/v3/module.seme" \
    -resource "$repo/modules/resource/v1/module.seme" \
    -durable-state "$repo/modules/durable-state/v1/module.seme" \
    -source-presentation "$repo/modules/source-presentation/v1/module.seme" \
    -ordered-transport "$repo/modules/ordered-transport/v1/module.seme" \
    -controlled-effects "$repo/modules/controlled-effects/v1/module.seme" \
    -target "$repo/modules/target/v1/module.seme" \
    -project-v8 "$repo/modules/project/v8/module.seme" \
    -project-v9 "$repo/modules/project/v9/module.seme" \
    -project-v10 "$repo/modules/project/v10/module.seme" \
    -project-v11 "$repo/modules/project/v11/module.seme" \
    -project-v12 "$repo/modules/project/v12/module.seme" \
    -project-v13 "$repo/modules/project/v13/module.seme" \
    -k0 "$repo/bootstrap/seme-k0-linux-amd64" \
    -g1-compiler "$repo/compiler/g1-compiler.k0"
}
printf 'UPB-11 project stage: two deterministic Project-v14 finalizations\n'
finalize "$work/final-a"
finalize "$work/final-b"
diff -ru "$work/final-a" "$work/final-b"
test -s "$work/final-a/patch-v1.seme"
test -s "$work/final-a/project-v14.seme"
printf 'Go UPB-11 full Project-v14 reconciliation and deterministic publication pass\n'
