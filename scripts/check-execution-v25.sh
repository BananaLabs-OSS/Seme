#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v25.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
 go run -buildvcs=false ./cmd/execution-module-v25 > "$work/module.g1" && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
cmp "$repo/modules/execution/v25/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v25" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v25/module.seme" "$work/module.seme"
(cd "$repo/fixtures/go-execution-v25" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test)

"$work/session" --module "$repo/modules/execution/v25/module.g1" \
  --project "$repo/fixtures/go-execution-v25" --package example.com/seme-collection-update-proof \
  --entry UpdateAndAppend --revision 1 --out "$work/go.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/go.g1" "$work/go.seme"
"$work/lower" "$work/go.seme" "$work/update.wasm" "$work/abi.json"
rg -q '"type": "slice:i64"' "$work/abi.json"
rg -q '"response_size": 0' "$work/abi.json"
node "$repo/reference/js/slice-result-v2-runner.mjs" "$work/update.wasm"

printf '%s\n' '/** @param {bigint[]} values @param {bigint} index @param {bigint} replacement @param {bigint} appended @returns {bigint[]} */' \
  'export function UpdateAndAppend(values, index, replacement, appended) { return values.with(Number(index), replacement).concat([appended]); }' > "$work/program.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/program.mjs" \
  --module "$repo/modules/execution/v25/module.g1" --package example.com/seme-collection-update-proof \
  --entry UpdateAndAppend --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/go.seme" "$work/js.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/go.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v25/module.g1" --package example.com/seme-collection-update-proof \
  --entry UpdateAndAppend --revision 1 --out "$work/relifted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/go.seme" "$work/relifted.seme"
node "$repo/reference/js/javascript-collection-update-native-runner.mjs" \
  "file://$work/projected.mjs" -7,0,42 1 9 100 > "$work/native.log"
rg -q '"result":\["-7","9","42","100"\]' "$work/native.log"
rg -q '"input":\["-7","0","42"\],"unchanged":true' "$work/native.log"

printf '%s\n' '/** @param {bigint[]} values @param {bigint} value @returns {bigint[]} */' \
  'export function Invalid(values, value) { values.push(value); return values; }' > "$work/invalid.mjs"
if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/invalid.mjs" \
  --module "$repo/modules/execution/v25/module.g1" --package example.com/seme-invalid-mutation \
  --revision 1 --out "$work/invalid.g1" > "$work/invalid.log" 2>&1; then
  echo "mutating JavaScript collection operation was accepted" >&2
  exit 1
fi

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/update.wasm" "$work/pulp/pure-function.wasm"
request=2000000003000000010000000000000009000000000000006400000000000000f9ffffffffffffff00000000000000002a00000000000000
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 \
  -request "$request" > "$work/pulp.log" 2>&1
rg -q '"response":"0800000004000000f9ffffffffffffff09000000000000002a000000000000006400000000000000"' "$work/pulp.log"

version=2
while [ "$version" -le 24 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo "Core Execution v25: immutable update and append produced alias-safe dynamic slice results across Go, JavaScript, Wasm, and Pulp"
