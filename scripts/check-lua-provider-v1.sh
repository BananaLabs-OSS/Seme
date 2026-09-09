#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-lua-provider.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

node --test "$repo/reference/lua/lua-provider.test.mjs"

mkdir "$work/source"
printf '%s\n' \
  '---@param value boolean' \
  '---@return boolean' \
  'local function Identity(value)' \
  '  return value' \
  'end' > "$work/source/identity.lua"
printf '%s\n' \
  '---@param enabled boolean' \
  '---@return boolean' \
  'function Run(enabled)' \
  '  return Identity(enabled)' \
  'end' > "$work/source/run.lua"

node "$repo/reference/lua/lua-provider-cli.mjs" \
  --source "$work/source/identity.lua" --source "$work/source/run.lua" \
  --module "$repo/modules/execution/v30/module.g1" \
  --package example.test/lua-uab-01 --revision 1 --entry Run --out "$work/lua.g1"
node "$repo/reference/lua/lua-provider-cli.mjs" \
  --source "$work/source/identity.lua" --source "$work/source/run.lua" \
  --module "$repo/modules/execution/v30/module.g1" \
  --package example.test/lua-uab-01 --revision 1 --entry Run --out "$work/lua-second.g1"
cmp "$work/lua.g1" "$work/lua-second.g1"

k0="$repo/bootstrap/seme-k0-linux-amd64"
"$k0" "$repo/compiler/g1-compiler.k0" "$work/lua.g1" "$work/lua.seme"
"$k0" "$repo/compiler/kernel-wire-validator.k0" "$work/lua.seme"
"$k0" "$repo/modules/foundation/v1/validator.k0" "$work/lua.seme"

node "$repo/reference/lua/lua-projector-cli.mjs" "$work/lua.g1" "$work/projected.lua"
node "$repo/reference/lua/lua-provider-cli.mjs" \
  --source "$work/projected.lua" --module "$repo/modules/execution/v30/module.g1" \
  --package example.test/lua-uab-01 --revision 1 --entry Run --out "$work/relifted.g1"
"$k0" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/lua.seme" "$work/relifted.seme"

# Reuse the existing embedded Lua without writing editor state into the repo.
lua_env="XDG_DATA_HOME=$work/nvim-data XDG_STATE_HOME=$work/nvim-state XDG_CACHE_HOME=$work/nvim-cache"
sed -n '1,$p' "$work/source/identity.lua" "$work/source/run.lua" > "$work/native.lua"
printf '%s\n' 'assert(Run(true) == true)' 'assert(Run(false) == false)' >> "$work/native.lua"
printf '%s\n' 'assert(Run(true) == true)' 'assert(Run(false) == false)' >> "$work/projected.lua"
env $lua_env nvim -l "$work/native.lua"
printf '%s\n' \
  '_G.Seme = dofile(assert(arg[1]))' \
  'dofile(assert(arg[2]))' > "$work/projected-runner.lua"
env $lua_env nvim -l "$work/projected-runner.lua" "$repo/reference/lua/seme-values.lua" "$work/projected.lua"

(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
"$work/lower" "$work/lua.seme" "$work/lua-a.wasm" "$work/lua-a.json"
"$work/lower" "$work/lua.seme" "$work/lua-b.wasm" "$work/lua-b.json"
cmp "$work/lua-a.wasm" "$work/lua-b.wasm"
cmp "$work/lua-a.json" "$work/lua-b.json"
rg -q '"fidelity": "exact"' "$work/lua-a.json"
rg -q '"type": "bool"' "$work/lua-a.json"
node "$repo/reference/js/pure-function-runner.mjs" "$work/lua-a.wasm" 00 01 > "$work/target.log"
rg -q '"request":"00","status":0,"response":"00"' "$work/target.log"
rg -q '"request":"01","status":0,"response":"01"' "$work/target.log"

if node "$repo/reference/lua/lua-provider-cli.mjs" \
  --source "$work/source/run.lua" --module "$repo/modules/execution/v30/module.g1" \
  --package example.test/lua-uab-01 --revision 1 --entry Run --out "$work/rejected.g1" \
  > "$work/rejection.log" 2>&1; then
  echo "Lua provider accepted an undeclared cross-file call" >&2
  exit 1
fi
rg -q 'lua.unknown_call:.*run.lua:4:1' "$work/rejection.log"

echo "Lua provider v1: UAB-01 direct lift/native/target/projection/rejection evidence passed"
