#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v19.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
 go run -buildvcs=false ./cmd/execution-module-v19 > "$work/module.g1" && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
cmp "$repo/modules/execution/v19/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v19" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v19/module.seme" "$work/module.seme"

(cd "$repo/fixtures/go-execution-v19" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test)
"$work/session" --module "$repo/modules/execution/v19/module.g1" \
  --project "$repo/fixtures/go-execution-v19" --package example.com/seme-choice-proof \
  --entry Choose --revision 1 --out "$work/go.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/go.g1" "$work/go.seme"
"$work/lower" "$work/go.seme" "$work/go.wasm" "$work/go-abi.json"
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/go.wasm" \
  f9ffffffffffffff2a0000000000000001 2a00000000000000
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/go.wasm" \
  f9ffffffffffffff2a0000000000000000 f9ffffffffffffff

printf '%s\n' '/** @param {bigint} original @param {bigint} replacement @param {boolean} enabled @returns {bigint} */' \
  'export function Choose(original, replacement, enabled) {' '  let result = original;' \
  '  if (enabled) {' '    result = replacement;' '  }' '  return result;' '}' > "$work/program.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/program.mjs" \
  --module "$repo/modules/execution/v19/module.g1" --package example.com/seme-choice-proof \
  --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/go.seme" "$work/js.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/go.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v19/module.g1" --package example.com/seme-choice-proof \
  --revision 1 --out "$work/relifted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/go.seme" "$work/relifted.seme"
node "$repo/reference/js/javascript-bigint-native-runner.mjs" "file://$work/projected.mjs" -7 42 true > "$work/native-true.log"
node "$repo/reference/js/javascript-bigint-native-runner.mjs" "file://$work/projected.mjs" -7 42 false > "$work/native-false.log"
rg -q '"result":"42"' "$work/native-true.log"
rg -q '"result":"-7"' "$work/native-false.log"

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/go.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 \
  -request f9ffffffffffffff2a0000000000000001 > "$work/pulp.log" 2>&1
rg -q '"response":"2a00000000000000"' "$work/pulp.log"

printf '%s\n' '/** @param {bigint} value @param {boolean} enabled @returns {bigint} */' \
  'export function Invalid(value, enabled) { let result = value; if (enabled) { result = value; } }' > "$work/invalid.mjs"
if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/invalid.mjs" \
  --module "$repo/modules/execution/v19/module.g1" --package example.com/seme-invalid-choice \
  --revision 1 --out "$work/invalid.g1" > "$work/invalid.log" 2>&1; then
  echo "non-total function was accepted" >&2
  exit 1
fi
rg -q 'javascript.block_not_total' "$work/invalid.log"

version=2
while [ "$version" -le 18 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo "Core Execution v19: signed-i64 mutation and one-sided Go/JavaScript choice ran natively, standalone, and through Pulp"
