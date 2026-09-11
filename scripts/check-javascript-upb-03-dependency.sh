#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
fixture=${JS_UPB_FIXTURE:-"$repo/fixtures/javascript-upb03-dependency"}; external="$repo/fixtures/javascript-upb03-external-checksum"
project=${JS_UPB_PROJECT:-example.test/javascript-upb03}; files=${JS_UPB_FILES:-application.js,math/sum.js}
root_module=${JS_UPB_ROOT_MODULE:-application.js}; local_file=${JS_UPB_LOCAL_FILE:-math/sum.js}
local_identity=${JS_UPB_LOCAL_IDENTITY:-example.test/javascript-upb03/math/sum}; local_from=${JS_UPB_LOCAL_FROM:-example.test/javascript-upb03/application}
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-upb03.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
(cd "$repo/reference/go" && for command in project-source-discover project-assemble source-inventory-emit package-detail-emit project-v3-compose dependency-emit project-v4-compose project-roundtrip-publish; do go build -buildvcs=false -o "$work/$command" "./cmd/$command"; done)
node --test "$repo/reference/js/javascript-npm-resolver.test.mjs"

(cd "$fixture" && node "$repo/reference/js/javascript-package-provider-cli.mjs" --files "$files" \
  --module "$repo/modules/execution/v35/module.g1" --package "$project" --entry Run --revision 1 --out "$work/execution.g1")
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/execution.g1" "$work/execution.seme"
"$work/project-assemble" -execution "$work/execution.seme" -manifest "$fixture/seme-package-manifest.json" \
  -execution-contract "$repo/modules/execution/v35/module.seme" -package-contract "$repo/modules/package/v1/module.seme" -project-contract "$repo/modules/project/v1/module.seme" -out "$work/project-v1.seme"
"$work/project-source-discover" -root "$fixture" -identity "$project" -language javascript -toolchain "$(node --version)" \
  -profile seme.javascript-upb03/v1 -semantic-revision seme.javascript-provider/uab11 -tracked-extensions .js -generated-header '// Code generated' -out "$work/snapshot.json"
"$work/source-inventory-emit" -snapshot "$work/snapshot.json" -project "$work/project-v1.seme" -execution-contract "$repo/modules/execution/v35/module.seme" \
  -package-contract "$repo/modules/package/v1/module.seme" -project-contract "$repo/modules/project/v2/module.seme" -out "$work/inventory-v2.seme"
(cd "$fixture" && node "$repo/reference/js/javascript-project-graph-cli.mjs" --files "$files" --snapshot "$work/snapshot.json" \
  --canonical-g1 "$work/execution.g1" --project "$project" --root-module "$root_module" --out "$work/package-graph.json")
"$work/package-detail-emit" -base "$work/project-v1.seme" -inventory "$work/inventory-v2.seme" -graph "$work/package-graph.json" \
  -execution-contract "$repo/modules/execution/v35/module.seme" -package-contract "$repo/modules/package/v1/module.seme" -project-contract "$repo/modules/project/v2/module.seme" -out "$work/package-v2.seme"
"$work/project-v3-compose" -execution-contract "$repo/modules/execution/v35/module.seme" -package-v1 "$repo/modules/package/v1/module.seme" \
  -package-v2 "$repo/modules/package/v2/module.seme" -project-v2 "$repo/modules/project/v2/module.seme" -project-v3 "$repo/modules/project/v3/module.seme" \
  -project "$work/project-v1.seme" -inventory "$work/inventory-v2.seme" -package-graph "$work/package-v2.seme" -out "$work/project-v3.seme"
resolve() { source=$1; output=$2; node "$repo/reference/js/javascript-npm-resolver-cli.mjs" --manifest "$source/package.json" --lock "$source/package-lock.json" \
  --local-root "$source" --local-files "$local_file" --external-root "$external" --external-files index.js,package.json \
  --local-identity "$local_identity" --local-from "$local_from" \
  --external-source file:../javascript-upb03-external-checksum --out "$output"; }
resolve "$fixture" "$work/closure-a.json"; resolve "$fixture" "$work/closure-b.json"; cmp "$work/closure-a.json" "$work/closure-b.json"
for name in a b; do "$work/dependency-emit" -closure "$work/closure-$name.json" -contract "$repo/modules/dependency/v1/module.seme" -out "$work/dependency-$name.seme"; done
cmp "$work/dependency-a.seme" "$work/dependency-b.seme"
compose() { "$work/project-v4-compose" -execution-contract "$repo/modules/execution/v35/module.seme" -package-v1 "$repo/modules/package/v1/module.seme" \
  -package-v2 "$repo/modules/package/v2/module.seme" -dependency-contract "$repo/modules/dependency/v1/module.seme" -project-v2 "$repo/modules/project/v2/module.seme" \
  -project-v3-contract "$repo/modules/project/v3/module.seme" -project-v4-contract "$repo/modules/project/v4/module.seme" -project-v1 "$work/project-v1.seme" \
  -inventory-v2 "$work/inventory-v2.seme" -package-graph-v2 "$work/package-v2.seme" -project-v3 "$work/project-v3.seme" -dependency "$work/dependency-a.seme" -out "$1"; }
compose "$work/project-v4-a.seme"; compose "$work/project-v4-b.seme"; cmp "$work/project-v4-a.seme" "$work/project-v4-b.seme"
if compose "$work/project-v4-a.seme" >"$work/collision.out" 2>"$work/collision.err"; then exit 1; fi
grep -q 'project_v4_compose.output_exists' "$work/collision.err"
cp "$work/dependency-a.seme" "$work/tampered.seme"; printf x >> "$work/tampered.seme"
if "$work/project-v4-compose" -execution-contract "$repo/modules/execution/v35/module.seme" -package-v1 "$repo/modules/package/v1/module.seme" -package-v2 "$repo/modules/package/v2/module.seme" -dependency-contract "$repo/modules/dependency/v1/module.seme" -project-v2 "$repo/modules/project/v2/module.seme" -project-v3-contract "$repo/modules/project/v3/module.seme" -project-v4-contract "$repo/modules/project/v4/module.seme" -project-v1 "$work/project-v1.seme" -inventory-v2 "$work/inventory-v2.seme" -package-graph-v2 "$work/package-v2.seme" -project-v3 "$work/project-v3.seme" -dependency "$work/tampered.seme" -out "$work/rejected.seme" >"$work/tampered.out" 2>"$work/tampered.err"; then exit 1; fi
test ! -e "$work/rejected.seme"
node "$repo/reference/js/javascript-module-projector-cli.mjs" "$work/execution.g1" "$work/package-graph.json" "$work/generated-modules"
"$work/project-roundtrip-publish" -root "$fixture" -snapshot "$work/snapshot.json" -dest "$work/projected-native" \
  -projected "application.js=$work/generated-modules/application.js" -projected "$local_file=$work/generated-modules/$local_file" \
  -tracked-extensions .js -generated-header '// Code generated'
cmp "$fixture/package.json" "$work/projected-native/package.json"
cmp "$fixture/package-lock.json" "$work/projected-native/package-lock.json"
cmp "$fixture/seme-package-manifest.json" "$work/projected-native/seme-package-manifest.json"
node --check "$work/projected-native/application.js"; node --check "$work/projected-native/$local_file"
resolve "$work/projected-native" "$work/projected-closure-a.json"
resolve "$work/projected-native" "$work/projected-closure-b.json"
cmp "$work/projected-closure-a.json" "$work/projected-closure-b.json"
grep -q '"Identity": "@seme/checksum"' "$work/projected-closure-a.json"
grep -q '"Integrity": "sha512-eMqVAv11saPfbJxoBzC2zTleDVMj7Ach1eexxYA6IwR0iYqDlW7InpfB+guJ4r2qm7RcKm50jIqrSWfXUfYsIA=="' "$work/projected-closure-a.json"
echo 'JavaScript UPB-03 dependency foundation: offline closure and Project-v4 authority pass'
