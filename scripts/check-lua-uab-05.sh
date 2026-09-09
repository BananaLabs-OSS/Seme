#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}; pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-lua-uab05.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
source="$repo/fixtures/lua-uab-05/program.lua"; vectors="$repo/fixtures/lua-uab-05/vectors.json"
node "$repo/reference/lua/lua-provider-cli.mjs" --source "$source" --module "$repo/modules/execution/v27/module.g1" --package example.test/lua-uab-05 --revision 1 --entry Dispatch --out "$work/source.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/source.g1" "$work/program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/program.seme"
node "$repo/reference/lua/lua-projector-cli.mjs" "$work/source.g1" "$work/projected.lua"
node "$repo/reference/lua/lua-provider-cli.mjs" --source "$work/projected.lua" --module "$repo/modules/execution/v27/module.g1" --package example.test/lua-uab-05 --revision 1 --entry Dispatch --out "$work/relift.g1"
cmp "$work/source.g1" "$work/relift.g1"
env XDG_DATA_HOME="$work/data-original" XDG_STATE_HOME="$work/state-original" XDG_CACHE_HOME="$work/cache-original" nvim -l "$repo/reference/lua/lua-uab-05-native.lua" "$source" "$vectors" 2> "$work/original.json"
env XDG_DATA_HOME="$work/data-projected" XDG_STATE_HOME="$work/state-projected" XDG_CACHE_HOME="$work/cache-projected" nvim -l "$repo/reference/lua/lua-uab-05-native.lua" "$work/projected.lua" "$vectors" 2> "$work/projected-native.json"
node "$repo/reference/js/json-equal.mjs" "$work/original.json" "$work/projected-native.json"
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/canonical-eval" ./cmd/canonical-eval && go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
node "$repo/reference/lua/lua-uab-05-vectors.mjs" "$vectors" > "$work/canonical-vectors.json"
"$work/canonical-eval" "$work/program.seme" "$work/canonical-vectors.json" --named-rejections > "$work/canonical.json"
node -e 'const fs=require("fs"),u=require("util"),a=JSON.parse(fs.readFileSync(process.argv[1])).valid,x=JSON.parse(fs.readFileSync(process.argv[2])),b={};for(const[k,v]of Object.entries(x.valid))b[k]=v.i64;if(!u.isDeepStrictEqual(a,b)||!u.isDeepStrictEqual(Object.keys(x.rejected).sort(),["amount-is-bool","missing-value","selector-is-i64"]))process.exit(1)' "$work/original.json" "$work/canonical.json"
"$work/lower" "$work/program.seme" "$work/program.wasm" "$work/abi.json"
node "$repo/reference/lua/lua-uab-05-wasm-runner.mjs" "$work/program.wasm" "$vectors" > "$work/wasm.json"
node -e 'const fs=require("fs"),u=require("util"),a=JSON.parse(fs.readFileSync(process.argv[1])).valid,b=JSON.parse(fs.readFileSync(process.argv[2])).valid;if(!u.isDeepStrictEqual(a,b))process.exit(1)' "$work/original.json" "$work/wasm.json"
for forgery in witness-contract dynamic-requirement; do
  node "$repo/reference/lua/lua-uab-05-forge.mjs" "$work/source.g1" "$work/forged-$forgery.g1" "$forgery"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/forged-$forgery.g1" "$work/forged-$forgery.seme"
  if "$work/lower" "$work/forged-$forgery.seme" "$work/forged-$forgery.wasm" "$work/forged-$forgery.json" > "$work/forged-$forgery.log" 2>&1; then echo "target accepted Lua UAB-05 forgery $forgery" >&2; exit 1; fi
  rg -q 'wasm.interface_' "$work/forged-$forgery.log"
done
test -f "$pulp_repo/go.mod"; git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pulp" "$work/pulp-pinned"; git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp/pulp.cell.toml"; cp "$work/program.wasm" "$work/pulp/pure-function.wasm"
node "$repo/reference/lua/lua-uab-05-wasm-runner.mjs" "$work/program.wasm" "$vectors" --tsv > "$work/requests.tsv"
: > "$work/pulp-audit.tsv"
tab=$(printf '\t'); while IFS="$tab" read -r kind name request response; do if [ "$kind" = valid ]; then (cd "$work/pulp" && "$work/pulp-runner" -manifest pulp.cell.toml -provider seme.function-v2 -request "$request") > "$work/pulp-$name.log" 2>&1; rg -q "\"response\":\"$response\"" "$work/pulp-$name.log"; elif (cd "$work/pulp" && "$work/pulp-runner" -manifest pulp.cell.toml -provider seme.function-v2 -request "$request") > "$work/pulp-malformed-$name.log" 2>&1; then echo "Pulp accepted malformed Lua UAB-05 request $name" >&2; exit 1; fi; printf '%s\t%s\n' "$kind" "$name" >> "$work/pulp-audit.tsv"; done < "$work/requests.tsv"
node -e 'const fs=require("fs"),names=p=>fs.readFileSync(p,"utf8").trim().split("\n").map(x=>x.split("\t").slice(0,2));require("assert").deepStrictEqual(names(process.argv[1]),names(process.argv[2]))' "$work/requests.tsv" "$work/pulp-audit.tsv"
sed 's/adjust = Offset_adjust/other = Offset_adjust/' "$source" > "$work/missing.lua"
if node "$repo/reference/lua/lua-provider-cli.mjs" --source "$work/missing.lua" --module "$repo/modules/execution/v27/module.g1" --package example.test/lua-uab-05-reject --revision 1 --entry Dispatch --out "$work/bad.g1" 2> "$work/bad.log"; then echo 'missing protocol method accepted' >&2; exit 1; fi
rg -q 'lua.implementation_method_set:' "$work/bad.log"
rg -q 'missing.lua:[1-9][0-9]*:[1-9][0-9]*' "$work/bad.log"
node -e 'const s=require(process.argv[1]),e=["lift","native_parity","target_parity","projection_round_trip","rejection"];require("assert").deepStrictEqual(s.languages.lua["UAB-05"],e)' "$repo/conformance/uab-v1/scorecard.json"
echo 'Lua UAB-05 complete acceptance: all five evidence classes pass'
