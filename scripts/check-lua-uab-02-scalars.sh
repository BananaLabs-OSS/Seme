#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}; pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-lua-scalars.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; GOCACHE="$work/go-cache"; export XDG_CACHE_HOME GOCACHE
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower && go build -buildvcs=false -o "$work/canonical-eval" ./cmd/canonical-eval)
mkdir "$work/pulp-pinned"; git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)

prove() {
  entry=$1
  node "$repo/reference/lua/lua-provider-cli.mjs" --source "$repo/fixtures/lua-uab-02-scalars/program.lua" --module "$repo/modules/execution/v32/module.g1" --package example.test/lua-scalars --revision 1 --entry "$entry" --out "$work/$entry.g1"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/$entry.g1" "$work/$entry.seme"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/$entry.seme"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/$entry.seme"
  node "$repo/reference/lua/lua-projector-cli.mjs" "$work/$entry.g1" "$work/$entry.lua"
  node "$repo/reference/lua/lua-provider-cli.mjs" --source "$work/$entry.lua" --module "$repo/modules/execution/v32/module.g1" --package example.test/lua-scalars --revision 1 --entry "$entry" --out "$work/$entry-relift.g1"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/$entry-relift.g1" "$work/$entry-relift.seme"
  cmp "$work/$entry.seme" "$work/$entry-relift.seme"
  env XDG_DATA_HOME="$work/nvim-data-$entry-original" XDG_STATE_HOME="$work/nvim-state-$entry-original" XDG_CACHE_HOME="$work/nvim-cache-$entry-original" nvim -l "$repo/reference/lua/lua-scalars-native.lua" "$repo/fixtures/lua-uab-02-scalars/program.lua" "$entry" > "$work/$entry-native.json"
  env XDG_DATA_HOME="$work/nvim-data-$entry-projected" XDG_STATE_HOME="$work/nvim-state-$entry-projected" XDG_CACHE_HOME="$work/nvim-cache-$entry-projected" nvim -l "$repo/reference/lua/lua-scalars-native.lua" "$work/$entry.lua" "$entry" > "$work/$entry-projected-native.json"
  node "$repo/reference/js/json-equal.mjs" "$work/$entry-native.json" "$work/$entry-projected-native.json"
  "$work/lower" "$work/$entry.seme" "$work/$entry.wasm" "$work/$entry.json"
  mkdir "$work/$entry-pulp"; cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/$entry-pulp/pulp.cell.toml"; cp "$work/$entry.wasm" "$work/$entry-pulp/pure-function.wasm"
}
prove ObserveI64
"$work/canonical-eval" "$work/ObserveI64.seme" "$repo/fixtures/lua-uab-02-scalars/i64-canonical.json" > "$work/i64-canonical.json"
node -e 'const fs=require("fs"),u=require("util"),a=JSON.parse(fs.readFileSync(process.argv[1])),x=JSON.parse(fs.readFileSync(process.argv[2])),b={zero:x.valid.zero.i64,wrap:x.valid.wrap.i64}; if(!u.isDeepStrictEqual(a,b))process.exit(1)' "$work/ObserveI64-native.json" "$work/i64-canonical.json"
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/ObserveI64.wasm" 0000000000000000 0100000000000000
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/ObserveI64.wasm" ffffffffffffff7f 0000000000000080
"$work/pulp-runner" -manifest "$work/ObserveI64-pulp/pulp.cell.toml" -provider seme.function-v2 -request 0000000000000000 -request ffffffffffffff7f > "$work/i64.log"
rg -q '"response":"0100000000000000"' "$work/i64.log"; rg -q '"response":"0000000000000080"' "$work/i64.log"
prove ObserveBoolean
"$work/canonical-eval" "$work/ObserveBoolean.seme" "$repo/fixtures/lua-uab-02-scalars/bool-canonical.json" > "$work/bool-canonical.json"
node -e 'const fs=require("fs"),u=require("util"),a=JSON.parse(fs.readFileSync(process.argv[1])),x=JSON.parse(fs.readFileSync(process.argv[2])),b={false_value:x.valid.false.bool??false,true_value:x.valid.true.bool??false}; if(!u.isDeepStrictEqual(a,b))process.exit(1)' "$work/ObserveBoolean-native.json" "$work/bool-canonical.json"
env XDG_DATA_HOME="$work/nvim-data-bool-malformed" XDG_STATE_HOME="$work/nvim-state-bool-malformed" XDG_CACHE_HOME="$work/nvim-cache-bool-malformed" \
  nvim -l "$repo/reference/lua/lua-boolean-boundary-native.lua"
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/ObserveBoolean.wasm" 00 00
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/ObserveBoolean.wasm" 01 01
"$work/pulp-runner" -manifest "$work/ObserveBoolean-pulp/pulp.cell.toml" -provider seme.function-v2 -request 00 -request 01 > "$work/bool.log"
rg -q '"response":"00"' "$work/bool.log"; rg -q '"response":"01"' "$work/bool.log"
prove ObserveText
"$work/canonical-eval" "$work/ObserveText.seme" "$repo/fixtures/lua-uab-02-scalars/text-canonical.json" > "$work/text-canonical.json"
node -e 'const fs=require("fs"),u=require("util"),a=JSON.parse(fs.readFileSync(process.argv[1])),x=JSON.parse(fs.readFileSync(process.argv[2])),b={empty:x.valid.empty.text,unicode:x.valid.unicode.text}; if(!u.isDeepStrictEqual(a,b))process.exit(1)' "$work/ObserveText-native.json" "$work/text-canonical.json"
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/ObserveText.wasm" 0800000000000000 21
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/ObserveText.wasm" 080000000a000000e4b896e7958cf09f9a80 e4b896e7958cf09f9a8021
"$work/pulp-runner" -manifest "$work/ObserveText-pulp/pulp.cell.toml" -provider seme.function-v2 -request 0800000000000000 -request 080000000a000000e4b896e7958cf09f9a80 > "$work/text.log"
rg -q '"response":"21"' "$work/text.log"; rg -q '"response":"e4b896e7958cf09f9a8021"' "$work/text.log"

if "$work/pulp-runner" -manifest "$work/ObserveI64-pulp/pulp.cell.toml" -provider seme.function-v2 -request 00 > /dev/null 2>&1; then echo "short i64 accepted" >&2; exit 1; fi
if "$work/pulp-runner" -manifest "$work/ObserveBoolean-pulp/pulp.cell.toml" -provider seme.function-v2 -request 02 > /dev/null 2>&1; then echo "noncanonical Boolean accepted" >&2; exit 1; fi
if "$work/pulp-runner" -manifest "$work/ObserveText-pulp/pulp.cell.toml" -provider seme.function-v2 -request 0800000002000000c328 > /dev/null 2>&1; then echo "invalid UTF-8 text accepted" >&2; exit 1; fi

# Unsupported numeric literals and malformed text adapter forms reject.
sed 's/Seme.i64_literal("1")/1/' "$repo/fixtures/lua-uab-02-scalars/program.lua" > "$work/bad.lua"
if node "$repo/reference/lua/lua-provider-cli.mjs" --source "$work/bad.lua" --module "$repo/modules/execution/v32/module.g1" --package example.test/lua-scalars --revision 1 --entry ObserveI64 --out "$work/bad.g1" 2> "$work/bad.log"; then exit 1; fi
rg -q 'lua.unsupported_expression' "$work/bad.log"
echo "Lua UAB-02 scalars: original/projected Lua, canonical Seme, Wasm, and pinned Pulp agree"
