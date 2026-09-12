#!/bin/sh
# Realize the neutral UPB12 rename as revision 2, then strip the bootstrap
# source and publish another closed source-free authority.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
destination=${1:?new revision authority destination required}
export_destination=${2:-};test ! -e "$destination";test -z "$export_destination"||test ! -e "$export_destination"
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-upb12-revision.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
cp -R "$repo/fixtures/javascript-upb05-configuration" "$work/bootstrap"
sed -i 's#example\.test/javascript-upb05#seme.upb12/service#g' "$work/bootstrap/configuration-selection.json" "$work/bootstrap/controlled-effects-selection.json" "$work/bootstrap/durable-selection.json" "$work/bootstrap/resources.json" "$work/bootstrap/transport-selection.json"
files=application.js,configuration.js,controlled.js,policy.js,state.js,transport.js
structured=configuration-selection.json,durable-selection.json,transport-selection.json,controlled-effects-selection.json
target=$(cd "$repo/reference/js" && node --input-type=module -e 'import{javascriptDeclarationIdentity as id}from"./javascript-provider.mjs";process.stdout.write(id("seme.upb12/service","InitializePolicy"))')
base_revision=$(node --input-type=module - "$repo/reference/js/javascript-project-reconcile.mjs" "$work/bootstrap" <<'NODE'
const{javascriptProjectRevision}=await import(process.argv[2]);process.stdout.write(javascriptProjectRevision({project:process.argv[3],files:["application.js","configuration.js","controlled.js","policy.js","state.js","transport.js"],structuredReferences:["configuration-selection.json","durable-selection.json","transport-selection.json","controlled-effects-selection.json"]}));
NODE
)
node "$repo/reference/js/javascript-project-reconcile-cli.mjs" --project "$work/bootstrap" --out "$work/result-source" --project-path seme.upb12/service --files "$files" --structured-references "$structured" --module "$repo/modules/execution/v36/module.g1" --entry Run --target "$target" --expected InitializePolicy --replacement BuildPolicy --revision 2 --base-revision "$base_revision" --native-runner "$repo/reference/js/javascript-upb11-native-runner.mjs"
JS_UPB_FIXTURE="$work/result-source" JS_UPB_PROJECT=seme.upb12/service JS_UPB_REVISION=2 JS_UPB_IDENTITY_EVIDENCE="$work/result-source/.seme-reconciliation-v1/identity-evidence.json" JS_UPB_NATIVE_RUNNER="$repo/reference/js/javascript-upb11-native-runner.mjs" JS_UPB09_EXPORT="$work/upb09" "$repo/scripts/check-javascript-upb-09-foundation.sh"
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/bundle" ./cmd/project-v12-bundle)
"$work/bundle" -bundle-v11 "$work/upb09/v11-bundle" -extension-v12 "$work/upb09/v12" -out "$work/base"
"$repo/scripts/place-upb12-seed.sh" "$work/base" "$work/result-source" "$work/placement"
mkdir "$work/export" "$work/export/inputs"; cp -R "$work/base" "$work/export/result-base"; cp -R "$work/placement" "$work/export/result-placement"
for name in configuration-selection.json durable-selection.json transport-selection.json controlled-effects-selection.json; do cp "$work/result-source/$name" "$work/export/inputs/$name"; done
"$repo/scripts/build-upb12-authority.sh" "$work/export" wasm32-pulp-upb12-host-v1 upb12 "$work/authority"
node --input-type=module - "$repo/reference/js/upb12-descriptor.mjs" "$repo/conformance/upb-v1/upb-12-project.json" "$work/authority" <<'NODE'
const{verifyUPB12Descriptor}=await import(process.argv[2]);const report=verifyUPB12Descriptor(process.argv[3],process.argv[4]);process.stdout.write(`UPB12 revision 2: ${report.packages.length} canonical packages, ${report.artifacts} source-free artifacts\n`);
NODE
parent=$(dirname "$destination"); test "$(cd "$parent" && pwd -P)" = "$parent"; mv -T -n "$work/authority" "$destination"; test ! -e "$work/authority"
if test -n "$export_destination"; then export_parent=$(dirname "$export_destination"); test "$(cd "$export_parent" && pwd -P)" = "$export_parent"; mv -T -n "$work/export" "$export_destination"; test ! -e "$work/export"; fi
