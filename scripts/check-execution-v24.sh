#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v24.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
 go run -buildvcs=false ./cmd/execution-module-v24 > "$work/module.g1" && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
cmp "$repo/modules/execution/v24/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v24" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v24/module.seme" "$work/module.seme"
(cd "$repo/fixtures/go-execution-v24" && go test -buildvcs=false ./...)
(cd "$repo/fixtures/go-execution-v24-boundary" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test)

lift() {
  project=$1 package=$2 entry=$3 prefix=$4
  "$work/session" --module "$repo/modules/execution/v24/module.g1" \
    --project "$project" --package "$package" --entry "$entry" --revision 1 --out "$prefix.g1"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$prefix.g1" "$prefix.seme"
  "$work/lower" "$prefix.seme" "$prefix.wasm" "$prefix.json"
}
lift "$repo/fixtures/go-execution-v24" example.com/seme-collection-query-proof LastOr "$work/last"
lift "$repo/fixtures/go-execution-v24-boundary" example.com/seme-dynamic-index-proof At "$work/at"
rg -q '"type": "slice:i64"' "$work/last.json"

empty=1000000000000000f7ffffffffffffff
many=1000000003000000f7fffffffffffffff9ffffffffffffff00000000000000002a00000000000000
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/last.wasm" "$empty" f7ffffffffffffff
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/last.wasm" "$many" 2a00000000000000

at_valid=10000000030000000200000000000000f9ffffffffffffff00000000000000002a00000000000000
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/at.wasm" "$at_valid" 2a00000000000000
for index in ffffffffffffffff 0300000000000000; do
  request=1000000003000000${index}f9ffffffffffffff00000000000000002a00000000000000
  if node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/at.wasm" "$request" 0000000000000000 > "$work/bounds.log" 2>&1; then
    echo "out-of-bounds dynamic slice index succeeded" >&2
    exit 1
  fi
  rg -q RuntimeError "$work/bounds.log"
done

printf '%s\n' '/** @param {bigint[]} values @param {bigint} fallback @returns {bigint} */' \
  'export function LastOr(values, fallback) { if (values.length <= 0n) return fallback; return Seme.index(values, values.length - 1); }' > "$work/program.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/program.mjs" \
  --module "$repo/modules/execution/v24/module.g1" --package example.com/seme-collection-query-proof \
  --entry LastOr --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/last.seme" "$work/js.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/last.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v24/module.g1" --package example.com/seme-collection-query-proof \
  --entry LastOr --revision 1 --out "$work/relifted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/last.seme" "$work/relifted.seme"
node "$repo/reference/js/javascript-collection-query-native-runner.mjs" "file://$work/projected.mjs" -9 "" > "$work/native.log"
node "$repo/reference/js/javascript-collection-query-native-runner.mjs" "file://$work/projected.mjs" -9 -7,0,42 >> "$work/native.log"
rg -q '"result":"-9"' "$work/native.log"
rg -q '"result":"42"' "$work/native.log"

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/last.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 \
  -request "$many" > "$work/pulp.log" 2>&1
rg -q '"response":"2a00000000000000"' "$work/pulp.log"

version=2
while [ "$version" -le 23 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo "Core Execution v24: neutral length and computed indexing agreed across Go, JavaScript, Wasm, and Pulp with strict bounds"
