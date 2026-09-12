#!/bin/sh
# Full JavaScript UPB11 reconciliation, rebuild, placement, and Project-v14 proof.
# This is an additive partition; only check-javascript-upb-11.sh may claim the cell.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
fixture="$repo/fixtures/javascript-upb05-configuration"
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-upb11-project.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"
export GOCACHE XDG_CACHE_HOME
files=application.js,configuration.js,controlled.js,policy.js,state.js,transport.js
structured=configuration-selection.json,durable-selection.json,transport-selection.json,controlled-effects-selection.json
project=example.test/javascript-upb05

(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/bundle" ./cmd/project-v12-bundle)
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/place" ./cmd/go-upb10-place)
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/finalize" ./cmd/project-v14-finalize)
(cd "$repo/reference/go" && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -buildvcs=false -o "$work/canonical-vm.wasm" ./cmd/canonical-wasm-cell)
cp "$repo/targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml" "$work/pulp.cell.toml"

target=$(cd "$repo/reference/js" && node --input-type=module -e 'import{javascriptDeclarationIdentity as id}from"./javascript-provider.mjs";process.stdout.write(id("example.test/javascript-upb05","InitializePolicy"))')
base_revision=$(node --input-type=module - "$repo/reference/js/javascript-project-reconcile.mjs" "$fixture" <<'NODE'
const{javascriptProjectRevision}=await import(process.argv[2]);
process.stdout.write(javascriptProjectRevision({project:process.argv[3],files:["application.js","configuration.js","controlled.js","policy.js","state.js","transport.js"],structuredReferences:["configuration-selection.json","durable-selection.json","transport-selection.json","controlled-effects-selection.json"]}));
NODE
)
node "$repo/reference/js/javascript-project-reconcile-cli.mjs" \
  --project "$fixture" --out "$work/result-source" --project-path "$project" \
  --files "$files" --structured-references "$structured" \
  --module "$repo/modules/execution/v36/module.g1" --entry Run \
  --target "$target" --expected InitializePolicy --replacement BuildPolicy --revision 2 --base-revision "$base_revision" \
  --native-runner "$repo/reference/js/javascript-upb11-native-runner.mjs"
node "$repo/reference/js/javascript-project-reconcile-verify-cli.mjs" \
  --project "$work/result-source" --project-path "$project" --files "$files" \
  --structured-references "$structured" --module "$repo/modules/execution/v36/module.g1" \
  --entry Run --native-runner "$repo/reference/js/javascript-upb11-native-runner.mjs" > "$work/reconciliation-report.json"

printf 'JavaScript UPB-11 project stage: independently rebuild both semantic revisions\n'
JS_UPB09_EXPORT="$work/prior-upb09" "$repo/scripts/check-javascript-upb-09-foundation.sh"
JS_UPB_FIXTURE="$work/result-source" JS_UPB_PROJECT="$project" JS_UPB_REVISION=2 \
  JS_UPB_IDENTITY_EVIDENCE="$work/result-source/.seme-reconciliation-v1/identity-evidence.json" \
  JS_UPB_NATIVE_RUNNER="$repo/reference/js/javascript-upb11-native-runner.mjs" \
  JS_UPB09_EXPORT="$work/result-upb09" "$repo/scripts/check-javascript-upb-09-foundation.sh"
for side in prior result; do
  "$work/bundle" -bundle-v11 "$work/$side-upb09/v11-bundle" \
    -extension-v12 "$work/$side-upb09/v12" -out "$work/$side-base"
done

common_place() {
  bundle=$1; selection_root=$2; shift 2
  "$work/place" -bundle "$bundle" \
    -selection "$selection_root/configuration-selection.json" \
    -durable-selection "$selection_root/durable-selection.json" \
    -transport-selection "$selection_root/transport-selection.json" \
    -effects-selection "$selection_root/controlled-effects-selection.json" \
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
    -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0" \
    -canonical-vm "$work/canonical-vm.wasm" -pulp-cell "$work/pulp.cell.toml" \
    -target-name wasm32-pulp-javascript-host-v1 -rule-namespace javascript-upb10 "$@"
}
printf 'JavaScript UPB-11 project stage: independently place both revisions\n'
common_place "$work/prior-base" "$fixture" -out "$work/prior-placement"
common_place "$work/result-base" "$work/result-source" -out "$work/result-placement"

finalize() {
  destination=$1
  reconciliation_root=${2:-"$work/result-source"}
  "$work/finalize" \
    -bundle "$work/prior-base" -placement "$work/prior-placement" \
    -selection "$fixture/configuration-selection.json" \
    -durable-selection "$fixture/durable-selection.json" \
    -transport-selection "$fixture/transport-selection.json" \
    -effects-selection "$fixture/controlled-effects-selection.json" \
    -result-bundle "$work/result-base" -result-placement "$work/result-placement" \
    -result-selection "$work/result-source/configuration-selection.json" \
    -reconciliation "$reconciliation_root" \
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
    -patch "$repo/modules/patch/v1/module.seme" \
    -language-service "$repo/modules/language-service/v1/module.seme" \
    -project-v14 "$repo/modules/project/v14/module.seme" \
    -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0" \
    -out "$destination"
}
printf 'JavaScript UPB-11 project stage: deterministic Project-v14 finalization\n'
finalize "$work/final-a"
finalize "$work/final-b"
diff -ru "$work/final-a" "$work/final-b"
if finalize "$work/final-a" >"$work/collision.out" 2>"$work/collision.err"; then
  echo 'JavaScript UPB-11 overwrote final authority' >&2; exit 1
fi
test ! -s "$work/collision.out"
test -s "$work/final-a/patch-v1.seme"
test -s "$work/final-a/project-v14.seme"

cp -R "$work/result-source" "$work/tampered-source"
printf x >> "$work/tampered-source/.seme-reconciliation-v1/identity-evidence.json"
if node "$repo/reference/js/javascript-project-reconcile-verify-cli.mjs" \
  --project "$work/tampered-source" --project-path "$project" --files "$files" \
  --structured-references "$structured" --module "$repo/modules/execution/v36/module.g1" \
  --entry Run --native-runner "$repo/reference/js/javascript-upb11-native-runner.mjs" \
  >"$work/tampered.out" 2>"$work/tampered.err"; then
  echo 'JavaScript UPB-11 accepted reconciled metadata tamper' >&2; exit 1
fi
test ! -s "$work/tampered.out"
if finalize "$work/tampered-final" "$work/tampered-source" >"$work/tampered-final.out" 2>"$work/tampered-final.err"; then
  echo 'JavaScript UPB-11 finalizer accepted reconciled metadata tamper' >&2; exit 1
fi
test ! -e "$work/tampered-final"
test ! -s "$work/tampered-final.out"
if test -n "${JS_UPB11_EXPORT:-}"; then
  test ! -e "$JS_UPB11_EXPORT"
  mkdir "$JS_UPB11_EXPORT"
  cp -R "$work/result-source" "$JS_UPB11_EXPORT/source"
  cp -R "$work/prior-base" "$JS_UPB11_EXPORT/prior-base"
  cp -R "$work/result-base" "$JS_UPB11_EXPORT/result-base"
  cp -R "$work/prior-placement" "$JS_UPB11_EXPORT/prior-placement"
  cp -R "$work/result-placement" "$JS_UPB11_EXPORT/result-placement"
  cp -R "$work/final-a" "$JS_UPB11_EXPORT/final"
fi
printf 'JavaScript UPB-11 full Project-v14 reconciliation and deterministic publication pass\n'
