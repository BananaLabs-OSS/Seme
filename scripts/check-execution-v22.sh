#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v22.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
 go run -buildvcs=false ./cmd/execution-module-v22 > "$work/module.g1" && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
cmp "$repo/modules/execution/v22/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v22" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v22/module.seme" "$work/module.seme"
(cd "$repo/fixtures/go-execution-v22" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test)

lift() {
  entry=$1
  output=$2
  "$work/session" --module "$repo/modules/execution/v22/module.g1" \
    --project "$repo/fixtures/go-execution-v22" --package example.com/seme-fold-proof \
    --entry "$entry" --revision 1 --out "$output.g1"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$output.g1" "$output.seme"
  "$work/lower" "$output.seme" "$output.wasm" "$output.json"
}
lift Sum "$work/sum"
lift SumEmpty "$work/empty"
node "$repo/reference/js/pure-function-runner.mjs" "$work/sum.wasm" \
  f9ffffffffffffff00000000000000002a00000000000000 \
  ffffffffffffff7f0100000000000000ffffffffffffffff > "$work/runtime.log"
rg -q '"response":"2300000000000000"' "$work/runtime.log"
rg -q '"response":"ffffffffffffff7f"' "$work/runtime.log"
node "$repo/reference/js/pure-function-runner.mjs" "$work/empty.wasm" "" > "$work/empty.log"
rg -q '"response":"0000000000000000"' "$work/empty.log"

printf '%s\n' '/** @param {bigint[3]} values @returns {bigint} */' \
  'export function Sum(values) { return values.reduce((total, value) => total + value, 0n); }' \
  '/** @param {bigint[0]} values @returns {bigint} */' \
  'function SumEmpty(values) { return values.reduce((total, value) => total + value, 0n); }' > "$work/program.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/program.mjs" \
  --module "$repo/modules/execution/v22/module.g1" --package example.com/seme-fold-proof \
  --entry Sum --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/sum.seme" "$work/js.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/sum.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/empty.g1" "$work/projected-empty.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v22/module.g1" --package example.com/seme-fold-proof \
  --entry Sum --revision 1 --out "$work/relifted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/sum.seme" "$work/relifted.seme"
node "$repo/reference/js/javascript-fold-native-runner.mjs" "file://$work/projected.mjs" -7,0,42 Sum > "$work/native.log"
node "$repo/reference/js/javascript-fold-native-runner.mjs" "file://$work/projected-empty.mjs" "" SumEmpty >> "$work/native.log"
rg -q '"result":"35"' "$work/native.log"
rg -q '"result":"0"' "$work/native.log"

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/sum.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 \
  -request f9ffffffffffffff00000000000000002a00000000000000 > "$work/pulp.log" 2>&1
rg -q '"response":"2300000000000000"' "$work/pulp.log"

printf '%s\n' '/** @param {bigint[3]} values @returns {bigint} */' \
  'export function Invalid(values) { return values.reduce((total, value) => value + total, 0n); }' > "$work/invalid.mjs"
if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/invalid.mjs" \
  --module "$repo/modules/execution/v22/module.g1" --package example.com/seme-invalid-fold \
  --revision 1 --out "$work/invalid.g1" > "$work/invalid.log" 2>&1; then
  echo "unsupported fold body was accepted" >&2
  exit 1
fi
rg -q 'javascript.fold_body' "$work/invalid.log"

version=2
while [ "$version" -le 21 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo "Core Execution v22: Go range and JavaScript reduce shared one typed deterministic fold through Wasm and Pulp"
