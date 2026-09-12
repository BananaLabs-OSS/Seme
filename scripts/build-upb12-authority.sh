#!/bin/sh
# Reopen one complete UPB11 export and publish its source-free Project-v13 seed.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
export_root=${1:?UPB11 export required};target=${2:?target name required};namespace=${3:?rule namespace required};destination=${4:?destination required}
test -d "$export_root/result-base";test -d "$export_root/result-placement";test -d "$export_root/inputs"
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-upb12-authority-build.XXXXXX");trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache";export GOCACHE
(cd "$repo/reference/go"&&go build -buildvcs=false -o "$work/build" ./cmd/upb12-authority-build)
"$work/build" \
  -bundle "$export_root/result-base" -placement "$export_root/result-placement" \
  -selection "$export_root/inputs/configuration-selection.json" \
  -durable-selection "$export_root/inputs/durable-selection.json" \
  -transport-selection "$export_root/inputs/transport-selection.json" \
  -effects-selection "$export_root/inputs/controlled-effects-selection.json" \
  -foundation "$repo/modules/foundation/v1/module.seme" -execution "$repo/modules/execution/v36/module.seme" \
  -package "$repo/modules/package/v4/module.seme" -dependency "$repo/modules/dependency/v1/module.seme" \
  -configuration "$repo/modules/configuration/v3/module.seme" -resource "$repo/modules/resource/v1/module.seme" \
  -durable-state "$repo/modules/durable-state/v1/module.seme" -source-presentation "$repo/modules/source-presentation/v1/module.seme" \
  -ordered-transport "$repo/modules/ordered-transport/v1/module.seme" -controlled-effects "$repo/modules/controlled-effects/v1/module.seme" \
  -project-v8 "$repo/modules/project/v8/module.seme" -project-v9 "$repo/modules/project/v9/module.seme" \
  -project-v10 "$repo/modules/project/v10/module.seme" -project-v11 "$repo/modules/project/v11/module.seme" \
  -project-v12 "$repo/modules/project/v12/module.seme" -target "$repo/modules/target/v1/module.seme" \
  -project-v13 "$repo/modules/project/v13/module.seme" -k0 "$repo/bootstrap/seme-k0-linux-amd64" \
  -g1-compiler "$repo/compiler/g1-compiler.k0" -target-name "$target" -rule-namespace "$namespace" -out "$destination"
node --input-type=module - "$repo/reference/js/upb12-authority.mjs" "$destination" <<'NODE'
const {loadUPB12Authority}=await import(process.argv[2]);const authority=loadUPB12Authority(process.argv[3]);process.stdout.write(`UPB12 source-free authority: ${authority.files.size} authenticated artifacts\n`);
NODE
