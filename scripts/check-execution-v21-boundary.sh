#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v21-boundary.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./goprovider ./wasmtarget && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
(cd "$repo/fixtures/go-execution-v21-boundary" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test)
"$work/session" --module "$repo/modules/execution/v21/module.g1" \
  --project "$repo/fixtures/go-execution-v21-boundary" --package example.com/seme-fixed-array-boundary-proof \
  --entry Pick --revision 1 --out "$work/go.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/go.g1" "$work/go.seme"
"$work/lower" "$work/go.seme" "$work/array.wasm" "$work/array-abi.json"
rg -q '"type": "fixed-array:i64:3"' "$work/array-abi.json"
rg -q '"size": 24' "$work/array-abi.json"
rg -q '"encoding": "packed-little-endian-twos-complement-i64"' "$work/array-abi.json"

valid_first=f9ffffffffffffff00000000000000002a000000000000000000000000000000
valid_last=f9ffffffffffffff00000000000000002a000000000000000200000000000000
node "$repo/reference/js/pure-function-runner.mjs" "$work/array.wasm" "$valid_first" "$valid_last" > "$work/valid.log"
rg -q '"response":"f9ffffffffffffff"' "$work/valid.log"
rg -q '"response":"2a00000000000000"' "$work/valid.log"
node "$repo/reference/js/pure-function-runner.mjs" "$work/array.wasm" \
  f9ffffffffffffff00000000000000002a0000000000000002000000000000 > "$work/truncated.log"
rg -q '"status":2,"response":""' "$work/truncated.log"
for request in \
  f9ffffffffffffff00000000000000002a00000000000000ffffffffffffffff \
  f9ffffffffffffff00000000000000002a000000000000000300000000000000
do
  if node "$repo/reference/js/pure-function-runner.mjs" "$work/array.wasm" "$request" > "$work/bounds.log" 2>&1; then
    echo "out-of-bounds array parameter read succeeded" >&2
    exit 1
  fi
  rg -q RuntimeError "$work/bounds.log"
done

printf '%s\n' '/** @param {bigint[3]} values @param {bigint} index @returns {bigint} */' \
  'export function Pick(values, index) { return Seme.index(values, index); }' > "$work/program.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/program.mjs" \
  --module "$repo/modules/execution/v21/module.g1" --package example.com/seme-fixed-array-boundary-proof \
  --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/go.seme" "$work/js.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/go.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v21/module.g1" --package example.com/seme-fixed-array-boundary-proof \
  --revision 1 --out "$work/relifted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/go.seme" "$work/relifted.seme"
node "$repo/reference/js/javascript-array-boundary-native-runner.mjs" "file://$work/projected.mjs" -7 0 42 2 > "$work/native.log"
rg -q '"result":"42"' "$work/native.log"

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/array.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 \
  -request "$valid_last" > "$work/pulp.log" 2>&1
rg -q '"response":"2a00000000000000"' "$work/pulp.log"
if "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 \
  -request f9ffffffffffffff00000000000000002a000000000000000300000000000000 > "$work/pulp-bounds.log" 2>&1; then
  echo "Pulp out-of-bounds array parameter read succeeded" >&2
  exit 1
fi

echo "Core Execution v21 boundary: canonical fixed arrays crossed the packed Wasm/Pulp ABI with strict bounds"
