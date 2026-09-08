#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v17.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
 go run -buildvcs=false ./cmd/execution-module-v17 > "$work/module.g1" && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
cmp "$repo/modules/execution/v17/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v17" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v17/module.seme" "$work/module.seme"

(cd "$repo/fixtures/go-execution-v17" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test)
"$work/session" --module "$repo/modules/execution/v17/module.g1" \
  --project "$repo/fixtures/go-execution-v17" --package example.com/seme-record-proof \
  --entry Label --revision 1 --out "$work/go.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/go.g1" "$work/go.seme"
"$work/lower" "$work/go.seme" "$work/go.wasm" "$work/go-abi.json"
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/go.wasm" \
  0800000007000000e99baaf09fa680 e99baaf09fa680

printf '%s\n' '/** @typedef {Object} Item' ' * @property {string} Name' \
  ' * @property {boolean} Enabled' ' */' \
  '/** @param {string} name @returns {string} */' \
  'export function Label(name) { const item = { Name: name, Enabled: true }; return item.Name; }' > "$work/program.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/program.mjs" \
  --module "$repo/modules/execution/v17/module.g1" --package example.com/seme-record-proof \
  --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/go.seme" "$work/js.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/go.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v17/module.g1" --package example.com/seme-record-proof \
  --revision 1 --out "$work/relifted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/go.seme" "$work/relifted.seme"
node "$repo/reference/js/javascript-native-runner.mjs" "file://$work/projected.mjs" '雪🦀' ignored Label > "$work/native.log"
rg -q '"result":"雪🦀"' "$work/native.log"

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/go.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 \
  -request 0800000007000000e99baaf09fa680 > "$work/pulp.log" 2>&1
rg -q '"response":"e99baaf09fa680"' "$work/pulp.log"

version=2
while [ "$version" -le 16 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo "Core Execution v17: compositional Go/JavaScript records ran natively, standalone, and through Pulp"
