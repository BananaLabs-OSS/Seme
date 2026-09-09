#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-lua-aggregate.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; GOCACHE="$work/go-cache"; export XDG_CACHE_HOME GOCACHE

node "$repo/reference/lua/lua-provider-cli.mjs" --source "$repo/fixtures/lua-uab-02-aggregate/program.lua" \
  --module "$repo/modules/execution/v32/module.g1" --package example.test/lua-aggregate \
  --revision 1 --entry Observe --out "$work/program.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/program.g1" "$work/program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/program.seme"
node "$repo/reference/lua/lua-projector-cli.mjs" "$work/program.g1" "$work/projected.lua"
node "$repo/reference/lua/lua-provider-cli.mjs" --source "$work/projected.lua" \
  --module "$repo/modules/execution/v32/module.g1" --package example.test/lua-aggregate \
  --revision 1 --entry Observe --out "$work/relifted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/program.seme" "$work/relifted.seme"

env XDG_DATA_HOME="$work/nvim-data-original" XDG_STATE_HOME="$work/nvim-state-original" XDG_CACHE_HOME="$work/nvim-cache-original" \
  nvim -l "$repo/reference/lua/lua-aggregate-native.lua" "$repo/fixtures/lua-uab-02-aggregate/program.lua" "$repo/fixtures/lua-uab-02-aggregate/vectors.json" > "$work/native.json"
env XDG_DATA_HOME="$work/nvim-data-projected" XDG_STATE_HOME="$work/nvim-state-projected" XDG_CACHE_HOME="$work/nvim-cache-projected" \
  nvim -l "$repo/reference/lua/lua-aggregate-native.lua" "$work/projected.lua" "$repo/fixtures/lua-uab-02-aggregate/vectors.json" > "$work/projected-native.json"
node "$repo/reference/js/json-equal.mjs" "$work/native.json" "$work/projected-native.json"

(cd "$repo/reference/go" && go test -buildvcs=false ./wasmtarget ./canonicaleval && go build -buildvcs=false -o "$work/lower" ./cmd/aggregate-wasm-lower && go build -buildvcs=false -o "$work/canonical-eval" ./cmd/canonical-eval)
"$work/canonical-eval" "$work/program.seme" "$repo/fixtures/lua-uab-02-aggregate/canonical-vectors.json" > "$work/canonical.json"
rg -q '"i64":"26"' "$work/canonical.json"; rg -q '"i64":"15"' "$work/canonical.json"; rg -q '"i64":"-9223372036854775808"' "$work/canonical.json"
node -e 'const fs=require("fs"),x=JSON.parse(fs.readFileSync(process.argv[1])),valid={}; for(const [k,v] of Object.entries(x.valid))valid[k]=v.i64; process.stdout.write(JSON.stringify({valid,malformed:x.malformed})+"\n")' "$work/canonical.json" > "$work/canonical-normalized.json"
node "$repo/reference/js/json-equal.mjs" "$work/native.json" "$work/canonical-normalized.json"
"$work/lower" "$work/program.seme" "$work/a.wasm" "$work/a.json"
"$work/lower" "$work/program.seme" "$work/b.wasm" "$work/b.json"
cmp "$work/a.wasm" "$work/b.wasm"; cmp "$work/a.json" "$work/b.json"
node "$repo/reference/js/aggregate-function-runner.mjs" "$work/a.wasm" "$repo/fixtures/lua-uab-02-aggregate/vectors.json" > "$work/wasm.json"
node "$repo/reference/js/json-equal.mjs" "$work/native.json" "$work/wasm.json"

# A same-signature nearby program lacking three required observations must not
# acquire executable authority merely because its package or entry name match.
sed 's@return Seme.add.*@return Seme.field(bounds, "bias")@' "$repo/fixtures/lua-uab-02-aggregate/program.lua" > "$work/nearby.lua"
node "$repo/reference/lua/lua-provider-cli.mjs" --source "$work/nearby.lua" \
  --module "$repo/modules/execution/v32/module.g1" --package example.test/lua-aggregate \
  --revision 1 --entry Observe --out "$work/nearby.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/nearby.g1" "$work/nearby.seme"
if "$work/lower" "$work/nearby.seme" "$work/bad.wasm" "$work/bad.json" 2> "$work/rejection.log"; then
  echo "aggregate lowerer accepted an uncertified nearby graph" >&2; exit 1
fi
rg -q 'wasm.aggregate_expression' "$work/rejection.log"

test -f "$pulp_repo/go.mod"; git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-aggregate-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/a.wasm" "$work/pulp/pure-function.wasm"
present=050000000000000002000000000000000700000000000000300000000300000048000000020000000900000000000000010000000000000002000000000000000300000000000000feffffffffffffff040000000000000009000000000000000b00000000000000
missing=050000000000000002000000000000000700000000000000300000000300000048000000020000000800000000000000010000000000000002000000000000000300000000000000feffffffffffffff040000000000000009000000000000000b00000000000000
wrapping=ffffffffffffff7f00000000000000000100000000000000300000000000000030000000000000000000000000000000
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-aggregate-v1 \
  -request "$present" -request "$missing" -request "$wrapping" > "$work/pulp.log" 2>&1
rg -q '"response":"1a00000000000000"' "$work/pulp.log"
rg -q '"response":"0f00000000000000"' "$work/pulp.log"
rg -q '"response":"0000000000000080"' "$work/pulp.log"
node "$repo/reference/js/aggregate-function-runner.mjs" "$work/a.wasm" "$repo/fixtures/lua-uab-02-aggregate/vectors.json" --tsv > "$work/requests.tsv"
tab=$(printf '\t')
while IFS="$tab" read -r kind name request response; do
  if [ "$kind" = malformed ] && "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-aggregate-v1 -request "$request" > "$work/pulp-malformed-$name.log" 2>&1; then
    echo "Pulp accepted malformed aggregate request $name" >&2; exit 1
  fi
done < "$work/requests.tsv"

echo "Lua aggregate UAB-02: original/projected Lua, canonical graph, Wasm, and pinned Pulp agree"
