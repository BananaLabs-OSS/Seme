#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v18.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
 go run -buildvcs=false ./cmd/execution-module-v18 > "$work/module.g1" && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
cmp "$repo/modules/execution/v18/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v18" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v18/module.seme" "$work/module.seme"

(cd "$repo/fixtures/go-execution-v18" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test)
"$work/session" --module "$repo/modules/execution/v18/module.g1" \
  --project "$repo/fixtures/go-execution-v18" --package example.com/seme-mutation-proof \
  --entry AppendOnce --revision 1 --out "$work/go.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/go.g1" "$work/go.seme"
"$work/lower" "$work/go.seme" "$work/go.wasm" "$work/go-abi.json"
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/go.wasm" \
  1100000003000000140000000400000001e99baaf09fa680 e99baaf09fa680
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/go.wasm" \
  1100000003000000140000000400000000e99baaf09fa680 e99baa

printf '%s\n' '/** @param {string} value @param {string} suffix @param {boolean} enabled @returns {string} */' \
  'export function AppendOnce(value, suffix, enabled) {' \
  '  let result = value;' '  let remaining = enabled;' \
  '  while (remaining) {' '    result = result + suffix;' '    remaining = false;' '  }' \
  '  return result;' '}' > "$work/program.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/program.mjs" \
  --module "$repo/modules/execution/v18/module.g1" --package example.com/seme-mutation-proof \
  --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/go.seme" "$work/js.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/go.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v18/module.g1" --package example.com/seme-mutation-proof \
  --revision 1 --out "$work/relifted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/go.seme" "$work/relifted.seme"
node "$repo/reference/js/javascript-mutation-native-runner.mjs" "file://$work/projected.mjs" '雪' '🦀' true > "$work/native-true.log"
node "$repo/reference/js/javascript-mutation-native-runner.mjs" "file://$work/projected.mjs" '雪' '🦀' false > "$work/native-false.log"
rg -q '"result":"雪🦀"' "$work/native-true.log"
rg -q '"result":"雪"' "$work/native-false.log"

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/go.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 \
  -request 1100000003000000140000000400000001e99baaf09fa680 > "$work/pulp.log" 2>&1
rg -q '"response":"e99baaf09fa680"' "$work/pulp.log"

printf '%s\n' '/** @param {string} value @param {boolean} enabled @returns {string} */' \
  'export function Spin(value, enabled) {' '  let remaining = enabled;' \
  '  while (remaining) {' '    remaining = remaining;' '  }' '  return value;' '}' > "$work/spin.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/spin.mjs" \
  --module "$repo/modules/execution/v18/module.g1" --package example.com/seme-loop-bound-proof \
  --revision 1 --out "$work/spin.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/spin.g1" "$work/spin.seme"
"$work/lower" "$work/spin.seme" "$work/spin.wasm" "$work/spin-abi.json"
if node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/spin.wasm" \
  0d000000010000000c0000000178 78 > "$work/spin.log" 2>&1; then
  echo "nonterminating loop did not trap" >&2
  exit 1
fi

version=2
while [ "$version" -le 17 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo "Core Execution v18: mutable Go/JavaScript places and bounded while ran natively, standalone, and through Pulp"
