#!/bin/sh
# Real source-free JavaScript UPB-10 target placement and adversaries.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
fixture=${JS_UPB_FIXTURE:-"$repo/fixtures/javascript-upb05-configuration"}
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-upb10-placement.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"
export GOCACHE XDG_CACHE_HOME

JS_UPB09_EXPORT="$work/upb09" "$repo/scripts/check-javascript-upb-09-foundation.sh"

(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/place" ./cmd/go-upb10-place)
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/report" ./cmd/go-upb10-report)
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/policy" ./cmd/go-upb10-policy-check)
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/closure" ./cmd/canonical-closure-check)
(cd "$repo/reference/go" && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -buildvcs=false -o "$work/canonical-vm.wasm" ./cmd/canonical-wasm-cell)
cp "$repo/targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml" "$work/pulp.cell.toml"

# Publish the cumulative Project-v12 authority in the standard closed bundle
# shape. The predecessor and extension were independently authenticated by the
# UPB-09 foundation gate above.
base="$work/base"
cp -R "$work/upb09/v11-bundle" "$base"
rm "$base/COMPLETE.sha256"
cp "$work/upb09/v12/controlled-effects-v1.seme" "$base/controlled-effects-v1.seme"
cp "$work/upb09/v12/project-v12.seme" "$base/project-v12.seme"
cp "$work/upb09/v12/controlled-replay-v1.json" "$base/controlled-replay-v1.json"
files='construction-v36.g1 execution-v36.seme project-base-v8.seme inventory-v8.seme package-detail-v4.seme package-v4.seme dependency-v1.seme configuration-v3.seme project-v8.seme resource-v1.seme project-v9.seme durable-state-v1.seme source-presentation-v1.seme project-v10.seme ordered-transport-v1.seme project-v11.seme controlled-effects-v1.seme project-v12.seme controlled-replay-v1.json'
printf 'seme-go-upb09-bundle-v1\n' > "$base/COMPLETE.sha256"
for f in $files; do printf '%s %s\n' "$f" "$(sha256sum "$base/$f" | cut -d ' ' -f 1)" >> "$base/COMPLETE.sha256"; done
for p in "$base"/blobs/*; do h=$(basename "$p"); printf 'blobs/%s %s\n' "$h" "$h" >> "$base/COMPLETE.sha256"; done

common() {
  "$@" \
    -bundle "$base" \
    -selection "$fixture/configuration-selection.json" \
    -durable-selection "$fixture/durable-selection.json" \
    -transport-selection "$fixture/transport-selection.json" \
    -effects-selection "$fixture/controlled-effects-selection.json" \
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
    -target-name wasm32-pulp-javascript-host-v1 \
    -rule-namespace javascript-upb10
}
place() { common "$work/place" -canonical-vm "$work/canonical-vm.wasm" -pulp-cell "$work/pulp.cell.toml" "$@"; }

place -out "$work/placement-a"
place -out "$work/placement-b"
diff -ru "$work/placement-a" "$work/placement-b"
common "$work/report" -placement "$work/placement-a" > "$work/report.json"
cmp "$work/report.json" "$work/placement-a/placement-report.json"

counts=$(node - "$work/report.json" <<'NODE'
const fs=require('fs'),r=JSON.parse(fs.readFileSync(process.argv[2]));
if(!r.executable||r.target_name!=='wasm32-pulp-javascript-host-v1'||r.boundary_count!==4||r.fidelities.native_island!==4||r.fidelities.impossible!==0||r.fidelities.adapted!==0||r.fidelities.emulated!==0||r.fidelities.embedded_runtime!==0||r.fidelities.refined!==0||r.requirement_count!==r.fidelities.exact+4)process.exit(1);
process.stdout.write(r.fidelities.exact+' '+r.requirement_count);
NODE
)
set -- $counts
exact=$1; requirements=$2
common "$work/policy" -expect-exact "$exact" -expect-impossible 4

if place -policy exact-only -out "$work/exact-output" > "$work/exact.out" 2> "$work/exact.err"; then
  echo 'JavaScript UPB-10 published an exact-only impossible plan' >&2; exit 1
fi
test ! -e "$work/exact-output"

cp -R "$work/placement-a" "$work/tampered"
printf x >> "$work/tampered/target-plan-v1.seme"
if common "$work/report" -placement "$work/tampered" > "$work/tampered.out" 2> "$work/tampered.err"; then
  echo 'JavaScript UPB-10 accepted a tampered plan' >&2; exit 1
fi
if place -out "$work/placement-a" > "$work/existing.out" 2> "$work/existing.err"; then
  echo 'JavaScript UPB-10 overwrote an existing placement' >&2; exit 1
fi
"$work/closure" "$work/placement-a/target-plan-v1.seme"
"$work/closure" "$work/placement-a/project-v13.seme"

if test -n "${JS_UPB10_EXPORT:-}"; then
  test ! -e "$JS_UPB10_EXPORT"
  mkdir "$JS_UPB10_EXPORT"
  cp -R "$work/upb09" "$JS_UPB10_EXPORT/upb09"
  cp -R "$base" "$JS_UPB10_EXPORT/base"
  cp -R "$work/placement-a" "$JS_UPB10_EXPORT/placement"
fi
printf 'JavaScript UPB-10 placement: %s requirements, %s exact, 4 explicit native islands, deterministic Project-v13 publication and adversaries pass\n' "$requirements" "$exact"
