#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
fixture="$repo/fixtures/javascript-upb03-dependency"; external="$repo/fixtures/javascript-upb03-external-checksum"
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-upb03.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
(cd "$repo/reference/go" && for command in project-source-discover project-assemble source-inventory-emit package-detail-emit project-v3-compose dependency-emit project-v4-compose; do go build -buildvcs=false -o "$work/$command" "./cmd/$command"; done)
node --test "$repo/reference/js/javascript-npm-resolver.test.mjs"

(cd "$fixture" && node "$repo/reference/js/javascript-package-provider-cli.mjs" --files application.js,math/sum.js \
  --module "$repo/modules/execution/v35/module.g1" --package example.test/javascript-upb03 --entry Run --revision 1 --out "$work/execution.g1")
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/execution.g1" "$work/execution.seme"
"$work/project-assemble" -execution "$work/execution.seme" -manifest "$fixture/seme-package-manifest.json" \
  -execution-contract "$repo/modules/execution/v35/module.seme" -package-contract "$repo/modules/package/v1/module.seme" -project-contract "$repo/modules/project/v1/module.seme" -out "$work/project-v1.seme"
"$work/project-source-discover" -root "$fixture" -identity example.test/javascript-upb03 -language javascript -toolchain "$(node --version)" \
  -profile seme.javascript-upb03/v1 -semantic-revision seme.javascript-provider/uab11 -tracked-extensions .js -generated-header '// Code generated' -out "$work/snapshot.json"
"$work/source-inventory-emit" -snapshot "$work/snapshot.json" -project "$work/project-v1.seme" -execution-contract "$repo/modules/execution/v35/module.seme" \
  -package-contract "$repo/modules/package/v1/module.seme" -project-contract "$repo/modules/project/v2/module.seme" -out "$work/inventory-v2.seme"
(cd "$fixture" && node "$repo/reference/js/javascript-project-graph-cli.mjs" --files application.js,math/sum.js --snapshot "$work/snapshot.json" \
  --canonical-g1 "$work/execution.g1" --project example.test/javascript-upb03 --root-module application.js --out "$work/package-graph.json")
"$work/package-detail-emit" -base "$work/project-v1.seme" -inventory "$work/inventory-v2.seme" -graph "$work/package-graph.json" \
  -execution-contract "$repo/modules/execution/v35/module.seme" -package-contract "$repo/modules/package/v1/module.seme" -project-contract "$repo/modules/project/v2/module.seme" -out "$work/package-v2.seme"
"$work/project-v3-compose" -execution-contract "$repo/modules/execution/v35/module.seme" -package-v1 "$repo/modules/package/v1/module.seme" \
  -package-v2 "$repo/modules/package/v2/module.seme" -project-v2 "$repo/modules/project/v2/module.seme" -project-v3 "$repo/modules/project/v3/module.seme" \
  -project "$work/project-v1.seme" -inventory "$work/inventory-v2.seme" -package-graph "$work/package-v2.seme" -out "$work/project-v3.seme"
resolve() { node "$repo/reference/js/javascript-npm-resolver-cli.mjs" --manifest "$fixture/package.json" --lock "$fixture/package-lock.json" \
  --local-root "$fixture" --local-files math/sum.js --external-root "$external" --external-files index.js,package.json \
  --local-identity example.test/javascript-upb03/math/sum --local-from example.test/javascript-upb03/application \
  --external-source file:../javascript-upb03-external-checksum --out "$1"; }
resolve "$work/closure-a.json"; resolve "$work/closure-b.json"; cmp "$work/closure-a.json" "$work/closure-b.json"
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
echo 'JavaScript UPB-03 dependency foundation: offline closure and Project-v4 authority pass'
