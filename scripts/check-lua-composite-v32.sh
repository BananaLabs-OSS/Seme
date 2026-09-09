#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-lua-composite-v32.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME
GOCACHE="$work/go-cache"; export GOCACHE

printf '%s\n' \
  '---@param value seme.option<seme.result<seme.bytes,seme.text>>' \
  '---@return boolean' \
  'function Check(value)' \
  '  return Seme.match_option(value, false, function(some) return Seme.match_result(some, function(ok) return Seme.bytes_equal(ok, Seme.bytes_literal("ok")) end, function(err) return Seme.text_equal(err, "bad") end) end)' \
  'end' > "$work/program.lua"

node "$repo/reference/lua/lua-provider-cli.mjs" --source "$work/program.lua" \
  --module "$repo/modules/execution/v32/module.g1" --package example.test/lua-composite \
  --revision 1 --entry Check --out "$work/program.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/program.g1" "$work/program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/program.seme"
node "$repo/reference/lua/lua-projector-cli.mjs" "$work/program.g1" "$work/projected.lua"
node "$repo/reference/lua/lua-provider-cli.mjs" --source "$work/projected.lua" \
  --module "$repo/modules/execution/v32/module.g1" --package example.test/lua-composite \
  --revision 1 --entry Check --out "$work/relifted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/program.seme" "$work/relifted.seme"

env XDG_DATA_HOME="$work/nvim-data" XDG_STATE_HOME="$work/nvim-state" XDG_CACHE_HOME="$work/nvim-cache" \
  nvim -l "$repo/reference/lua/lua-composite-v32-native.lua"

if [ -n "${SEME_COMPOSITE_WASM_LOWER:-}" ]; then cp "$SEME_COMPOSITE_WASM_LOWER" "$work/lower"
else (cd "$repo/reference/go" && go test -buildvcs=false ./wasmtarget && go build -buildvcs=false -o "$work/lower" ./cmd/composite-wasm-lower); fi
"$work/lower" "$work/program.seme" "$work/a.wasm" "$work/a.json"
"$work/lower" "$work/program.seme" "$work/b.wasm" "$work/b.json"
cmp "$work/a.wasm" "$work/b.wasm"; cmp "$work/a.json" "$work/b.json"
node "$repo/reference/js/composite-function-runner.mjs" "$work/a.wasm"

test -f "$pulp_repo/go.mod"; git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-composite-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/a.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-composite-v1 \
  -request 00000000000000000000 -request 01000a000000020000006f6b \
  -request 01010a00000003000000626164 > "$work/pulp.log" 2>&1
rg -q '"request":"00000000000000000000","response":"00"' "$work/pulp.log"
rg -q '"request":"01000a000000020000006f6b","response":"01"' "$work/pulp.log"
rg -q '"request":"01010a00000003000000626164","response":"01"' "$work/pulp.log"

echo "Lua composite v32: direct total matches agree natively, canonically, through Wasm and pinned Pulp"
