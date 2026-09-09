#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-lua-compound.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
k0="$repo/bootstrap/seme-k0-linux-amd64"

node --test "$repo/reference/lua/lua-provider.test.mjs"
env XDG_DATA_HOME="$work/nvim-data" XDG_STATE_HOME="$work/nvim-state" XDG_CACHE_HOME="$work/nvim-cache" \
  nvim -l "$repo/reference/lua/seme-values.test.lua"

printf '%s\n' \
  '---@param value seme.i64' \
  '---@return seme.option<seme.i64>' \
  'function Some(value)' \
  '  return Seme.some(value)' \
  'end' > "$work/option.lua"
printf '%s\n' \
  '---@param value seme.text' \
  '---@return seme.result<seme.i64,seme.text>' \
  'function Reject(value)' \
  '  return Seme.err(value)' \
  'end' > "$work/result.lua"
printf '%s\n' '---@param value seme.i64' '---@return seme.i64' 'function I64(value)' '  return value' 'end' > "$work/i64.lua"
printf '%s\n' '---@param value boolean' '---@return boolean' 'function Boolean(value)' '  return value' 'end' > "$work/boolean.lua"
printf '%s\n' '---@param value seme.text' '---@return seme.text' 'function Text(value)' '  return value' 'end' > "$work/text.lua"
printf '%s\n' '---@param value seme.bytes' '---@return seme.bytes' 'function Bytes(value)' '  return value' 'end' > "$work/bytes.lua"

printf '%s\n' '---@param a seme.i64' '---@param b seme.i64' '---@param c seme.i64' '---@return seme.array<seme.i64,3>' 'function Array(a, b, c)' '  return Seme.array(a, b, c)' 'end' > "$work/array.lua"
printf '%s\n' '---@param value seme.slice<seme.text>' '---@return seme.slice<seme.text>' 'function Slice(value)' '  return value' 'end' > "$work/slice.lua"
printf '%s\n' '---@param values seme.map<seme.i64,seme.bytes>' '---@param key seme.i64' '---@param value seme.bytes' '---@return seme.map<seme.i64,seme.bytes>' 'function Map(values, key, value)' '  return Seme.map_update(values, key, value)' 'end' > "$work/map.lua"
printf '%s\n' '---@class Bounds' '---@field minimum seme.i64' '---@field maximum seme.i64' '' '---@param value Bounds' '---@return seme.i64' 'function Record(value)' '  return Seme.field(value, "minimum")' 'end' > "$work/record.lua"

for name in i64 boolean text bytes option result array slice map record; do
  case "$name" in
    i64) entry=I64 ;;
    boolean) entry=Boolean ;;
    text) entry=Text ;;
    bytes) entry=Bytes ;;
    option) entry=Some ;;
    result) entry=Reject ;;
    array) entry=Array ;;
    slice) entry=Slice ;;
    map) entry=Map ;;
    record) entry=Record ;;
  esac
  node "$repo/reference/lua/lua-provider-cli.mjs" \
    --source "$work/$name.lua" --module "$repo/modules/execution/v31/module.g1" \
    --package "example.test/lua-$name" --revision 1 --entry "$entry" --out "$work/$name.g1"
  "$k0" "$repo/compiler/g1-compiler.k0" "$work/$name.g1" "$work/$name.seme"
  "$k0" "$repo/compiler/kernel-wire-validator.k0" "$work/$name.seme"
  "$k0" "$repo/modules/foundation/v1/validator.k0" "$work/$name.seme"
  node "$repo/reference/lua/lua-projector-cli.mjs" "$work/$name.g1" "$work/$name-projected.lua"
  node "$repo/reference/lua/lua-provider-cli.mjs" \
    --source "$work/$name-projected.lua" --module "$repo/modules/execution/v31/module.g1" \
    --package "example.test/lua-$name" --revision 1 --entry "$entry" --out "$work/$name-relifted.g1"
  "$k0" "$repo/compiler/g1-compiler.k0" "$work/$name-relifted.g1" "$work/$name-relifted.seme"
  cmp "$work/$name.seme" "$work/$name-relifted.seme"
done

echo "Lua compound foundation: native adapters and canonical tagged/collection type round trips passed"
