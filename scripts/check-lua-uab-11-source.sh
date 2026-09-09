#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/lua-uab11-source.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
module="$repo/modules/execution/v35/module.g1"
package=seme.uab11/application
cd "$repo"

node reference/lua/lua-provider-cli.mjs \
  --source fixtures/lua-uab-11/policy.lua --source fixtures/lua-uab-11/application.lua \
  --module "$module" --package "$package" --revision 1 --entry Apply --out "$work/program.g1"
bootstrap/seme-k0-linux-amd64 compiler/g1-compiler.k0 "$work/program.g1" "$work/program.seme"
bootstrap/seme-k0-linux-amd64 compiler/kernel-wire-validator.k0 "$work/program.seme"
test "$(rg -c '^en [0-9a-f]{32} 0000000000000000000000000000a035 ' "$work/program.g1")" -eq 1

node reference/lua/lua-projector-cli.mjs "$work/program.g1" "$work/projected.lua"
node reference/lua/lua-provider-cli.mjs --source "$work/projected.lua" --module "$module" \
  --package "$package" --revision 1 --entry Apply --out "$work/relift.g1"
cmp "$work/program.g1" "$work/relift.g1"

node reference/lua/lua-uab-11-corpus.mjs >"$work/corpus.json"
for source in fixtures/lua-uab-11/application.lua "$work/projected.lua"; do
  label=$(basename "$source" .lua)
  env XDG_DATA_HOME="$work/$label-data" XDG_STATE_HOME="$work/$label-state" XDG_CACHE_HOME="$work/$label-cache" \
    nvim -l reference/lua/lua-uab-11-native.lua reference/lua/seme-values.lua \
    fixtures/lua-uab-11/policy.lua "$source" "$work/corpus.json" >"$work/$label.json"
done
node -e 'const fs=require("fs"),assert=require("assert");assert.deepStrictEqual(JSON.parse(fs.readFileSync(process.argv[1])),JSON.parse(fs.readFileSync(process.argv[2])))' "$work/application.json" "$work/projected.json"

# The independent JavaScript oracle emits the same language-neutral recursive
# boundary values. This proves the Lua graph, not a source-language evaluator.
node reference/js/javascript-uab-11-canonical-corpus.mjs \
  "$repo/fixtures/javascript-uab-11/application.js" "$work/requests.jsonl" "$work/expected.jsonl"
(cd reference/go && go build -buildvcs=false -o "$work/canonical-observe" ./cmd/canonical-observe)
"$work/canonical-observe" "$work/program.seme" <"$work/requests.jsonl" >"$work/canonical.jsonl"
node reference/js/javascript-uab-11-jsonl-equal.mjs "$work/expected.jsonl" "$work/canonical.jsonl"

sed -n '17p' "$work/requests.jsonl" | sed 's/"observability.log"/"denied"/' >"$work/denied.jsonl"
if "$work/canonical-observe" "$work/program.seme" <"$work/denied.jsonl" >"$work/denied.out" 2>"$work/denied.log"; then exit 1; fi
test ! -s "$work/denied.out"; rg -q 'canonicaleval.effect_denied' "$work/denied.log"
sed -n '17p' "$work/requests.jsonl" | sed 's/"Scale":{"kind":"bool","bool":true}/"Scale":{"kind":"i64","i64":"1"}/' >"$work/malformed.jsonl"
if "$work/canonical-observe" "$work/program.seme" <"$work/malformed.jsonl" >"$work/malformed.out" 2>"$work/malformed.log"; then exit 1; fi
test ! -s "$work/malformed.out"; rg -q 'canonicaleval' "$work/malformed.log"

for forgery in effect-authority erase-stateful-call; do
  node reference/js/javascript-uab-11-forge.mjs "$work/program.g1" "$forgery" "$work/$forgery.g1"
  if node reference/lua/lua-projector-cli.mjs "$work/$forgery.g1" "$work/$forgery.lua" >"$work/$forgery.log" 2>&1; then exit 1; fi
  rg -q 'lua_projection\.' "$work/$forgery.log"; test ! -e "$work/$forgery.lua"
done

printf '%s\n' '---@param values seme.slice<seme.i64>' '---@return seme.i64' 'function Raw(values)' ' return values[1]' 'end' >"$work/raw.lua"
if node reference/lua/lua-provider-cli.mjs --source "$work/raw.lua" --module "$module" \
  --package seme.uab11/reject --revision 1 --entry Raw --out "$work/raw.g1" >"$work/raw.log" 2>&1; then exit 1; fi
rg -q 'lua.unsupported_expression:.*raw.lua:4:1' "$work/raw.log"; test ! -e "$work/raw.g1"

echo 'Lua UAB-11 source: two-file lift, canonical execution, native projection parity, byte re-lift, denial, malformed values, and graph forgeries pass'
