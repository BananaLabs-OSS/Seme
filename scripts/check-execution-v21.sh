#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v21.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
 go run -buildvcs=false ./cmd/execution-module-v21 > "$work/module.g1" && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
cmp "$repo/modules/execution/v21/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v21" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v21/module.seme" "$work/module.seme"

(cd "$repo/fixtures/go-execution-v21" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test)
"$work/session" --module "$repo/modules/execution/v21/module.g1" \
  --project "$repo/fixtures/go-execution-v21" --package example.com/seme-fixed-array-proof \
  --entry Pick --revision 1 --out "$work/go.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/go.g1" "$work/go.seme"
"$work/lower" "$work/go.seme" "$work/array.wasm" "$work/array-abi.json"
rg -q '"fidelity": "exact"' "$work/array-abi.json"
node "$repo/reference/js/pure-function-runner.mjs" "$work/array.wasm" \
  f9ffffffffffffff00000000000000002a000000000000000000000000000000 \
  f9ffffffffffffff00000000000000002a000000000000000200000000000000 > "$work/valid.log"
rg -q '"response":"f9ffffffffffffff"' "$work/valid.log"
rg -q '"response":"2a00000000000000"' "$work/valid.log"
for request in \
  f9ffffffffffffff00000000000000002a00000000000000ffffffffffffffff \
  f9ffffffffffffff00000000000000002a000000000000000300000000000000
do
  if node "$repo/reference/js/pure-function-runner.mjs" "$work/array.wasm" "$request" > "$work/bounds.log" 2>&1; then
    echo "out-of-bounds index succeeded" >&2
    exit 1
  fi
  rg -q RuntimeError "$work/bounds.log"
done

printf '%s\n' '/** @param {bigint} first @param {bigint} second @param {bigint} third @param {bigint} index @returns {bigint} */' \
  'export function Pick(first, second, third, index) { return [first, second, third][index]; }' > "$work/program.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/program.mjs" \
  --module "$repo/modules/execution/v21/module.g1" --package example.com/seme-fixed-array-proof \
  --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/go.seme" "$work/js.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/go.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v21/module.g1" --package example.com/seme-fixed-array-proof \
  --revision 1 --out "$work/relifted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/go.seme" "$work/relifted.seme"
node "$repo/reference/js/javascript-array-native-runner.mjs" "file://$work/projected.mjs" -7 0 42 2 > "$work/native.log"
rg -q '"result":"42"' "$work/native.log"

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/array.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 \
  -request f9ffffffffffffff00000000000000002a000000000000000200000000000000 > "$work/pulp.log" 2>&1
rg -q '"response":"2a00000000000000"' "$work/pulp.log"
if "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 \
  -request f9ffffffffffffff00000000000000002a000000000000000300000000000000 > "$work/pulp-bounds.log" 2>&1; then
  echo "Pulp out-of-bounds index succeeded" >&2
  exit 1
fi

printf '%s\n' '/** @param {bigint} value @param {bigint} index @returns {bigint} */' \
  'export function Invalid(value, index) { return [value, , value][index]; }' > "$work/invalid.mjs"
if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/invalid.mjs" \
  --module "$repo/modules/execution/v21/module.g1" --package example.com/seme-invalid-array \
  --revision 1 --out "$work/invalid.g1" > "$work/invalid.log" 2>&1; then
  echo "sparse fixed array was accepted" >&2
  exit 1
fi
rg -q 'javascript.fixed_array_shape' "$work/invalid.log"

version=2
while [ "$version" -le 20 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo "Core Execution v21: fixed i64 arrays and indexed reads agreed across Go, JavaScript, Wasm, and Pulp"
