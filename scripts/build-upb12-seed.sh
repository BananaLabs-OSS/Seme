#!/bin/sh
# Construct the neutral cumulative service once, remove its native source, and
# publish the authenticated source-free authority consumed by every projector.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.."&&pwd);destination=${1:?new authority destination required};export_destination=${2:-};test ! -e "$destination";test -z "$export_destination"||test ! -e "$export_destination"
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-upb12-seed.XXXXXX");trap 'rm -rf "$work"' EXIT HUP INT TERM;GOCACHE="$work/go-cache";export GOCACHE
cp -R "$repo/fixtures/javascript-upb05-configuration" "$work/bootstrap"
sed -i 's#example\.test/javascript-upb05#seme.upb12/service#g' "$work/bootstrap/configuration-selection.json" "$work/bootstrap/controlled-effects-selection.json" "$work/bootstrap/durable-selection.json" "$work/bootstrap/resources.json" "$work/bootstrap/transport-selection.json"
JS_UPB_FIXTURE="$work/bootstrap" JS_UPB_PROJECT=seme.upb12/service JS_UPB09_EXPORT="$work/upb09" "$repo/scripts/check-javascript-upb-09-foundation.sh"
(cd "$repo/reference/go"&&go build -buildvcs=false -o "$work/bundle" ./cmd/project-v12-bundle)
"$work/bundle" -bundle-v11 "$work/upb09/v11-bundle" -extension-v12 "$work/upb09/v12" -out "$work/base"
"$repo/scripts/place-upb12-seed.sh" "$work/base" "$work/bootstrap" "$work/placement"
mkdir "$work/export" "$work/export/inputs";cp -R "$work/base" "$work/export/result-base";cp -R "$work/placement" "$work/export/result-placement"
for name in configuration-selection.json durable-selection.json transport-selection.json controlled-effects-selection.json;do cp "$work/bootstrap/$name" "$work/export/inputs/$name";done
"$repo/scripts/build-upb12-authority.sh" "$work/export" wasm32-pulp-upb12-host-v1 upb12 "$work/authority"
node --input-type=module - "$repo/reference/js/upb12-descriptor.mjs" "$repo/conformance/upb-v1/upb-12-project.json" "$work/authority" <<'NODE'
const{verifyUPB12Descriptor}=await import(process.argv[2]);const report=verifyUPB12Descriptor(process.argv[3],process.argv[4]);process.stdout.write(`UPB12 neutral seed: ${report.packages.length} canonical packages, ${report.artifacts} source-free artifacts\n`);
NODE
parent=$(dirname "$destination");test "$(cd "$parent"&&pwd -P)" = "$parent";mv -T -n "$work/authority" "$destination";test ! -e "$work/authority"
if test -n "$export_destination";then export_parent=$(dirname "$export_destination");test "$(cd "$export_parent"&&pwd -P)" = "$export_parent";mv -T -n "$work/export" "$export_destination";test ! -e "$work/export";fi
