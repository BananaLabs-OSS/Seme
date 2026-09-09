#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v23.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
 go run -buildvcs=false ./cmd/execution-module-v23 > "$work/module.g1" && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
cmp "$repo/modules/execution/v23/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v23" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v23/module.seme" "$work/module.seme"
(cd "$repo/fixtures/go-execution-v23" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test)

"$work/session" --module "$repo/modules/execution/v23/module.g1" \
  --project "$repo/fixtures/go-execution-v23" --package example.com/seme-slice-proof \
  --entry Sum --revision 1 --out "$work/go.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/go.g1" "$work/go.seme"
"$work/lower" "$work/go.seme" "$work/sum.wasm" "$work/abi.json"
rg -q '"contract": "seme.pure-abi/v2"' "$work/abi.json"
rg -q '"type": "slice:i64"' "$work/abi.json"
rg -q '"encoding": "u32le-offset-u32le-element-count/packed-i64"' "$work/abi.json"
rg -q '"maximum_request_size": 7160' "$work/abi.json"

empty=0800000000000000
three=0800000003000000f9ffffffffffffff00000000000000002a00000000000000
five=080000000500000001000000000000000200000000000000030000000000000004000000000000000500000000000000
overflow=0800000003000000ffffffffffffff7f0100000000000000ffffffffffffffff
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/sum.wasm" "$empty" 0000000000000000
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/sum.wasm" "$three" 2300000000000000
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/sum.wasm" "$five" 0f00000000000000
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/sum.wasm" "$overflow" ffffffffffffff7f
node "$repo/reference/js/slice-abi-v2-runner.mjs" "$work/sum.wasm"

printf '%s\n' '/** @param {bigint[]} values @returns {bigint} */' \
  'export function Sum(values) { return values.reduce((total, value) => BigInt.asIntN(64, total + value), 0n); }' > "$work/program.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/program.mjs" \
  --module "$repo/modules/execution/v23/module.g1" --package example.com/seme-slice-proof \
  --entry Sum --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/go.seme" "$work/js.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/go.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v23/module.g1" --package example.com/seme-slice-proof \
  --entry Sum --revision 1 --out "$work/relifted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/go.seme" "$work/relifted.seme"
node "$repo/reference/js/javascript-fold-native-runner.mjs" "file://$work/projected.mjs" -7,0,42 Sum

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/sum.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 \
  -request "$three" > "$work/pulp.log" 2>&1
rg -q '"response":"2300000000000000"' "$work/pulp.log"

version=2
while [ "$version" -le 22 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo "Core Execution v23: Go and JavaScript runtime-sized i64 slices shared one bounded dynamic fold through Wasm and Pulp"
