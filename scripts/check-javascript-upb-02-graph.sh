#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
fixture="$repo/fixtures/javascript-upb02-modules"
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-upb02-graph.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
(cd "$repo/reference/go" &&
  go build -buildvcs=false -o "$work/discover" ./cmd/project-source-discover &&
  go build -buildvcs=false -o "$work/assemble" ./cmd/project-assemble &&
  go build -buildvcs=false -o "$work/inventory" ./cmd/source-inventory-emit &&
  go build -buildvcs=false -o "$work/detail" ./cmd/package-detail-emit &&
  go build -buildvcs=false -o "$work/compose" ./cmd/project-v3-compose &&
  go build -buildvcs=false -o "$work/report" ./cmd/project-v3-report)
node --test "$repo/reference/js/javascript-project-graph.test.mjs"

build() {
  source=$1
  destination=$2
  mkdir "$destination"
  (cd "$source" && node "$repo/reference/js/javascript-package-provider-cli.mjs" \
    --files application.js,math/sum.js --module "$repo/modules/execution/v35/module.g1" \
    --package example.test/javascript-upb02 --entry Run --revision 1 --out "$destination/execution.g1")
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$destination/execution.g1" "$destination/execution.seme"
  "$work/assemble" -execution "$destination/execution.seme" -manifest "$fixture/seme-package-manifest.json" \
    -execution-contract "$repo/modules/execution/v35/module.seme" -package-contract "$repo/modules/package/v1/module.seme" \
    -project-contract "$repo/modules/project/v1/module.seme" -out "$destination/project-v1.seme"
  "$work/discover" -root "$source" -identity example.test/javascript-upb02 -language javascript \
    -toolchain "$(node --version)" -profile seme.javascript-upb02/v1 -semantic-revision seme.javascript-provider/uab11 \
    -tracked-extensions .js -generated-header '// Code generated' -out "$destination/snapshot.json"
  "$work/inventory" -snapshot "$destination/snapshot.json" -project "$destination/project-v1.seme" \
    -execution-contract "$repo/modules/execution/v35/module.seme" -package-contract "$repo/modules/package/v1/module.seme" \
    -project-contract "$repo/modules/project/v2/module.seme" -out "$destination/inventory-v2.seme"
  (cd "$source" && node "$repo/reference/js/javascript-project-graph-cli.mjs" --files application.js,math/sum.js \
    --snapshot "$destination/snapshot.json" --canonical-g1 "$destination/execution.g1" \
    --project example.test/javascript-upb02 --root-module application.js --out "$destination/package-graph.json")
  "$work/detail" -base "$destination/project-v1.seme" -inventory "$destination/inventory-v2.seme" \
    -graph "$destination/package-graph.json" -execution-contract "$repo/modules/execution/v35/module.seme" \
    -package-contract "$repo/modules/package/v1/module.seme" -project-contract "$repo/modules/project/v2/module.seme" \
    -out "$destination/package-v2.seme"
  "$work/compose" -execution-contract "$repo/modules/execution/v35/module.seme" \
    -package-v1 "$repo/modules/package/v1/module.seme" -package-v2 "$repo/modules/package/v2/module.seme" \
    -project-v2 "$repo/modules/project/v2/module.seme" -project-v3 "$repo/modules/project/v3/module.seme" \
    -project "$destination/project-v1.seme" -inventory "$destination/inventory-v2.seme" \
    -package-graph "$destination/package-v2.seme" -out "$destination/project-v3.seme"
}
build "$fixture" "$work/a"
build "$fixture" "$work/b"
for file in execution.g1 execution.seme project-v1.seme snapshot.json inventory-v2.seme package-graph.json package-v2.seme project-v3.seme; do cmp "$work/a/$file" "$work/b/$file"; done
"$work/report" --execution "$repo/modules/execution/v35/module.seme" --package-v1 "$repo/modules/package/v1/module.seme" \
  --package-v2 "$repo/modules/package/v2/module.seme" --project-v2-contract "$repo/modules/project/v2/module.seme" \
  --project-v3-contract "$repo/modules/project/v3/module.seme" --project "$work/a/project-v1.seme" \
  --inventory "$work/a/inventory-v2.seme" --package-graph "$work/a/package-v2.seme" \
  --composed "$work/a/project-v3.seme" > "$work/report.json"
grep -q '"Visibility":"public"' "$work/report.json"
grep -q '"Visibility":"package"' "$work/report.json"
grep -q '"Requested":"./math/sum.js"' "$work/report.json"
grep -q '"Alias":"Sum"' "$work/report.json"
node "$repo/reference/js/javascript-module-projector-cli.mjs" "$work/a/execution.g1" "$work/a/package-graph.json" "$work/projected-modules"
node --check "$work/projected-modules/application.js"
node --check "$work/projected-modules/math/sum.js"
node "$repo/reference/js/javascript-uab-01-vectors.mjs" > "$work/vectors.json"
node "$repo/reference/js/javascript-package-native-runner.mjs" "$fixture/application.js" "$work/vectors.json" > "$work/native-original.json"
node "$repo/reference/js/javascript-package-native-runner.mjs" "$work/projected-modules/application.js" "$work/vectors.json" > "$work/native-projected.json"
cmp "$work/native-original.json" "$work/native-projected.json"
(cd "$work/projected-modules" && node "$repo/reference/js/javascript-package-provider-cli.mjs" \
  --files application.js,math/sum.js --module "$repo/modules/execution/v35/module.g1" \
  --package example.test/javascript-upb02 --entry Run --revision 1 --out "$work/relifted.g1")
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/a/execution.seme" "$work/relifted.seme"
"$work/assemble" -execution "$work/relifted.seme" -manifest "$fixture/seme-package-manifest.json" \
  -execution-contract "$repo/modules/execution/v35/module.seme" -package-contract "$repo/modules/package/v1/module.seme" \
  -project-contract "$repo/modules/project/v1/module.seme" -out "$work/relifted-project.seme"
cmp "$work/a/project-v1.seme" "$work/relifted-project.seme"
build "$work/projected-modules" "$work/projected-a"
build "$work/projected-modules" "$work/projected-b"
cmp "$work/a/project-v1.seme" "$work/projected-a/project-v1.seme"
cmp "$work/projected-a/project-v3.seme" "$work/projected-b/project-v3.seme"
if build "$fixture" "$work/a" >"$work/collision.out" 2>"$work/collision.err"; then exit 1; fi
test -s "$work/a/project-v3.seme"
echo 'JavaScript UPB-02 graph foundation: deterministic Package-v2 and Project-v3 authority pass'
