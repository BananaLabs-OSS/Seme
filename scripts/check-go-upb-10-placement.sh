#!/bin/sh
# Real source-free Go UPB-10 placement, fidelity, publication, and adversaries.
# This is an additive partition; only check-go-upb-10.sh may claim the cell.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb10-placement.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"
export GOCACHE XDG_CACHE_HOME

"$repo/scripts/materialize-go-upb09-fixture.sh" "$work/source"
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/build09" ./cmd/go-upb09-build)
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/place" ./cmd/go-upb10-place)
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/report" ./cmd/go-upb10-report)
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/policy" ./cmd/go-upb10-policy-check)
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/closure" ./cmd/canonical-closure-check)
(cd "$repo/reference/go" && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -buildvcs=false -o "$work/canonical-vm.wasm" ./cmd/canonical-wasm-cell)
cp "$repo/targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml" "$work/pulp.cell.toml"

proxy="$repo/fixtures/go-upb03-offline-proxy"
printf 'UPB-10 placement stage: build authenticated Project-v12 base\n'
"$work/build09" \
  -project "$work/source" -proxy "$proxy" \
  -module example.test/go-uab-11 \
  -package example.test/go-uab-11/streamservice -entry DispatchControlled \
  -revision 1 -out "$work/base" \
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

common() {
  "$@" \
    -bundle "$work/base" \
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

place() {
  common "$work/place" -canonical-vm "$work/canonical-vm.wasm" -pulp-cell "$work/pulp.cell.toml" "$@"
}

printf 'UPB-10 placement stage: two independent placements\n'
place -out "$work/placement-a"
place -out "$work/placement-b"
diff -ru "$work/placement-a" "$work/placement-b"
printf 'UPB-10 placement stage: independent source-free report\n'
common "$work/report" -placement "$work/placement-a" > "$work/report.json"
cmp "$work/report.json" "$work/placement-a/placement-report.json"
rg -q '"requirement_count": 92' "$work/report.json"
rg -q '"exact": 88' "$work/report.json"
rg -q '"native_island": 4' "$work/report.json"
rg -q '"impossible": 0' "$work/report.json"
printf 'UPB-10 placement stage: exact-only fidelity policy\n'
common "$work/policy" -expect-exact 88 -expect-impossible 4

# Exact-only may produce diagnostic meaning internally, but it must never
# publish a target placement or Project-v13 artifact.
printf 'UPB-10 placement stage: atomic publication adversaries\n'
if place -policy exact-only -out "$work/exact-output" > "$work/exact.out" 2> "$work/exact.err"; then
  echo 'UPB-10 published an exact-only impossible plan' >&2; exit 1
fi
test ! -e "$work/exact-output"

# Directory integrity and the independently regenerated plan both reject
# tampering. No replacement output is produced on either path.
cp -R "$work/placement-a" "$work/tampered"
printf x >> "$work/tampered/target-plan-v1.seme"
if common "$work/report" -placement "$work/tampered" > "$work/tampered.out" 2> "$work/tampered.err"; then
  echo 'UPB-10 report accepted a tampered plan' >&2; exit 1
fi
if place -out "$work/placement-a" > "$work/existing.out" 2> "$work/existing.err"; then
  echo 'UPB-10 overwrote an existing placement' >&2; exit 1
fi

printf 'UPB-10 placement stage: closed canonical envelopes\n'
"$work/closure" "$work/placement-a/target-plan-v1.seme"
"$work/closure" "$work/placement-a/project-v13.seme"

printf 'Go UPB-10 placement: 92 requirements, 88 exact, 4 native islands, exact-only impossible, deterministic source-free publication and adversaries pass\n'
