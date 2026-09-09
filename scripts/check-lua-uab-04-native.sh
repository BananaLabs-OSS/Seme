#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-lua-uab04.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
module="$repo/modules/execution/v34/module.g1"
source="$repo/fixtures/lua-uab-04/program.lua"
vectors="$repo/fixtures/lua-uab-04/vectors.json"
for entry in ConstructSlice Append Update Remove Length Index Traverse EmptyMap Insert Lookup RemoveMap; do
  node "$repo/reference/lua/lua-provider-cli.mjs" --source "$source" --module "$module" --package "example.test/lua-uab04-$entry" --revision 1 --entry "$entry" --out "$work/$entry.g1"
  node "$repo/reference/lua/lua-projector-cli.mjs" "$work/$entry.g1" "$work/$entry.lua"
  node "$repo/reference/lua/lua-provider-cli.mjs" --source "$work/$entry.lua" --module "$module" --package "example.test/lua-uab04-$entry" --revision 1 --entry "$entry" --out "$work/$entry-relift.g1"
  cmp "$work/$entry.g1" "$work/$entry-relift.g1"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/$entry.g1" "$work/$entry.seme"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/$entry-relift.g1" "$work/$entry-relift.seme"
  cmp "$work/$entry.seme" "$work/$entry-relift.seme"
  env XDG_DATA_HOME="$work/data-original-$entry" XDG_STATE_HOME="$work/state-original-$entry" XDG_CACHE_HOME="$work/cache-original-$entry" nvim -l "$repo/reference/lua/lua-uab-04-native.lua" "$source" "$vectors" "$entry" > "$work/$entry-original.json"
  env XDG_DATA_HOME="$work/data-projected-$entry" XDG_STATE_HOME="$work/state-projected-$entry" XDG_CACHE_HOME="$work/cache-projected-$entry" nvim -l "$repo/reference/lua/lua-uab-04-native.lua" "$work/$entry.lua" "$vectors" "$entry" > "$work/$entry-projected.json"
  node -e 'const fs=require("fs"),assert=require("assert");assert.deepStrictEqual(JSON.parse(fs.readFileSync(process.argv[1])),JSON.parse(fs.readFileSync(process.argv[2])))' "$work/$entry-original.json" "$work/$entry-projected.json"
done
for mutation in raw-table raw-index one-based-literal nil-deletion bad-fold; do
  case "$mutation" in
    raw-table) sed 's/Seme.slice(first, second)/{first, second}/' "$source" ;;
    raw-index) sed 's/Seme.index_zero(values, index)/values[index]/' "$source" ;;
    one-based-literal) sed 's/Seme.index_zero(values, index)/Seme.index_zero(values, 1)/' "$source" ;;
    nil-deletion) sed 's/return Seme.map_remove(values, key)/values[key] = nil/' "$source" ;;
    bad-fold) sed 's/Seme.add(accumulator, element)/Seme.add(element, accumulator)/' "$source" ;;
  esac > "$work/bad.lua"
  if node "$repo/reference/lua/lua-provider-cli.mjs" --source "$work/bad.lua" --module "$module" --package example.test/lua-uab04-reject --revision 1 --entry ConstructSlice --out "$work/bad.g1" 2> "$work/bad.log"; then
    echo "Lua UAB-04 mutation accepted: $mutation" >&2
    exit 1
  fi
  rg -q 'lua\.' "$work/bad.log"
done
echo "Lua UAB-04 native/projected: one corpus preserves immutable collection operations and rejection observations"
