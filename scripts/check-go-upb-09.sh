#!/bin/sh
# Cumulative Go UPB-09 seven-class authority gate.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb09.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"
export GOCACHE XDG_CACHE_HOME

# Preserve the complete claimed predecessor and reproduce the two additive
# neutral contracts independently.
"$repo/scripts/check-go-upb-08.sh"
"$repo/scripts/check-go-upb-09-fixture.sh"
"$repo/scripts/check-controlled-effects-v1.sh"
"$repo/scripts/check-project-contract-v12.sh"
"$repo/scripts/check-go-upb-09-runtime.sh"
"$repo/scripts/check-go-upb-09-placement.sh"

# Exercise the presently implemented producer/consumer, real materialized
# authority, replay, closed-world bundle, report, and command-boundary evidence.
(cd "$repo/reference/go" && go test -p=1 -count=1 \
  ./gocontrolledeffectsmanifest ./gocontrolledeffectsadapter \
  ./controlledeffectsinstance ./controlledeffectsruntime ./goupb09replay \
  ./projectv12instance ./goupb09pipeline ./goupb09bundle ./goupb09report \
  ./goupb09cmdload ./cmd/go-upb09-build ./cmd/go-upb09-project \
  ./cmd/go-upb09-report)

# Two independent producer executions must publish byte-identical closed-world
# bundles from the same authenticated ordinary source and pinned inputs.
proxy="$repo/fixtures/go-upb03-offline-proxy"
"$repo/scripts/materialize-go-upb09-fixture.sh" "$work/source"
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/build" ./cmd/go-upb09-build)
build() {
  destination=$1
  "$work/build" \
    -project "$work/source" -proxy "$proxy" \
    -module example.test/go-uab-11 \
    -package example.test/go-uab-11/streamservice -entry DispatchControlled \
    -revision 1 -out "$destination" \
    -dependency example.test/seme/checksum -version v1.2.3 \
    -local-from example.test/go-uab-11/application -local-to example.test/go-uab-11/model \
    -selection "$work/source/configuration-selection.json" \
    -resources "$work/source/resources.json" -resource-owner example.test/go-uab-11/service \
    -durable-selection "$work/source/durable-selection.json" \
    -transport-selection "$repo/fixtures/go-upb08-transport-overlay/transport-selection.json" \
    -effects-selection "$work/source/controlled-effects-selection.json" \
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
build "$work/bundle-a"
build "$work/bundle-b"
diff -ru "$work/bundle-a" "$work/bundle-b"

# Project the authenticated source-free bundle back to ordinary Go, build it
# offline, re-lift it, and require an exact canonical construction/execution
# fixed point while retaining honest source-provenance differences.
"$repo/scripts/check-go-upb-09-fixed-point.sh"

printf 'Go UPB-09 seven-class authority gate passes\n'
