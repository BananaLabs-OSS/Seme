#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/lua-uab-10.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
module="$repo/modules/execution/v30/module.g1"
source="$repo/fixtures/lua-uab-10/program.lua"
formatted="$repo/fixtures/lua-uab-10/formatted.lua"
vectors="$repo/fixtures/lua-uab-10/vectors.json"
identity=80112233445566778899aabbccddeeff

lift() {
  node "$repo/reference/lua/lua-provider-cli.mjs" --source "$1" --module "$module" \
    --package example.test/lua-uab-10 --revision 1 --entry "$2" --out "$3"
}

lift "$source" Identity "$work/original.g1"
lift "$formatted" Identity "$work/formatted.g1"
cmp "$work/original.g1" "$work/formatted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/original.g1" "$work/original.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/original.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/original.seme"

node "$repo/reference/lua/lua-uab-10-rename.mjs" "$work/original.g1" "$work/renamed.g1" "$identity" Identity Preserve
rg -q "^en $identity 00000000000000000000000000009011 " "$work/renamed.g1"
node "$repo/reference/lua/lua-projector-cli.mjs" "$work/renamed.g1" "$work/projected.lua"
rg -q "^---@seme-id $identity$" "$work/projected.lua"
rg -q '^function Preserve\(value\)$' "$work/projected.lua"
lift "$work/projected.lua" Preserve "$work/relift.g1"
cmp "$work/renamed.g1" "$work/relift.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/renamed.g1" "$work/renamed.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/renamed.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/renamed.seme"

env XDG_DATA_HOME="$work/native-data" XDG_STATE_HOME="$work/native-state" XDG_CACHE_HOME="$work/native-cache" \
  nvim -l "$repo/reference/lua/lua-uab-10-native.lua" "$source" Identity >"$work/native.json"
env XDG_DATA_HOME="$work/projected-data" XDG_STATE_HOME="$work/projected-state" XDG_CACHE_HOME="$work/projected-cache" \
  nvim -l "$repo/reference/lua/lua-uab-10-native.lua" "$work/projected.lua" Preserve >"$work/projected.json"
cmp "$work/native.json" "$work/projected.json"

(cd "$repo/reference/go" && go test -buildvcs=false ./canonicaleval ./wasmtarget && \
  go build -buildvcs=false -o "$work/eval" ./cmd/canonical-eval && \
  go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
"$work/eval" "$work/renamed.seme" "$vectors" --named-rejections >"$work/canonical.json"
node -e 'const x=require(process.argv[1]);if(x.valid.false.kind!=="bool"||x.valid.false.bool===true||x.valid.true.bool!==true||x.malformed!==3||Object.keys(x.rejected).sort().join(",")!=="extra,missing,wrong-type")process.exit(1)' "$work/canonical.json"
"$work/lower" "$work/renamed.seme" "$work/program.wasm" "$work/abi.json"
node "$repo/reference/lua/lua-uab-10-wasm-runner.mjs" "$work/program.wasm" >"$work/wasm.log"
rg -q '"name":"false","response":"00"' "$work/wasm.log"
rg -q '"name":"true","response":"01"' "$work/wasm.log"
test "$(rg -c '"rejected":true' "$work/wasm.log")" -eq 3

mkdir "$work/pulp-pinned" "$work/pulp"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/program.wasm" "$work/pulp/pure-function.wasm"
for row in false:00:00 true:01:01; do
  name=${row%%:*}; rest=${row#*:}; request=${rest%%:*}; response=${rest#*:}
  "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 -request "$request" >"$work/pulp-$name.log" 2>&1
  rg -q "\"response\":\"$response\"" "$work/pulp-$name.log"
done
for row in extra:0001 invalid-bool:02; do
  name=${row%%:*}; request=${row#*:}
  if "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 -request "$request" >"$work/pulp-$name.log" 2>&1; then
    echo "Pulp accepted malformed identity-proof request $name" >&2
    exit 1
  fi
done

sed "s/$identity/00000000000000000000000000009011/" "$source" >"$work/forged.lua"
if lift "$work/forged.lua" Identity "$work/forged.g1" 2>"$work/forged.log"; then
  echo 'Lua provider accepted a forged schema identity' >&2
  exit 1
fi
rg -q 'lua.invalid_semantic_identity:.*forged.lua:1:1' "$work/forged.log"
cp "$source" "$work/duplicate.lua"
sed 's/function Identity/local function Other/' "$source" >>"$work/duplicate.lua"
if lift "$work/duplicate.lua" Identity "$work/duplicate.g1" 2>"$work/duplicate.log"; then
  echo 'Lua provider accepted duplicate semantic identities' >&2
  exit 1
fi
rg -q 'lua.duplicate_semantic_identity:.*duplicate.lua:[1-9][0-9]*:' "$work/duplicate.log"

echo 'Lua UAB-10 complete acceptance: stable identity and all five evidence classes pass'
