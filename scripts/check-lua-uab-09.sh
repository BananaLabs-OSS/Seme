#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.."&&pwd);pulp_repo=${PULP_REPO:-"$repo/../Pulp"};commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001;work=$(mktemp -d "${TMPDIR:-/tmp}/lua09.XXXXXX");trap 'rm -rf "$work"' EXIT;GOCACHE="$work/go";XDG_CACHE_HOME="$work/cache";export GOCACHE XDG_CACHE_HOME;source="$repo/fixtures/lua-uab-09/program.lua";vectors="$repo/fixtures/lua-uab-09/vectors.json";module="$repo/modules/execution/v20/module.g1"
node "$repo/reference/lua/lua-provider-cli.mjs" --source "$source" --module "$module" --package example.test/lua-uab-09 --revision 1 --entry Observe --out "$work/a.g1";"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/a.g1" "$work/a.seme";node "$repo/reference/lua/lua-projector-cli.mjs" "$work/a.g1" "$work/projected.lua";node "$repo/reference/lua/lua-provider-cli.mjs" --source "$work/projected.lua" --module "$module" --package example.test/lua-uab-09 --revision 1 --entry Observe --out "$work/b.g1";cmp "$work/a.g1" "$work/b.g1"
env XDG_DATA_HOME="$work/d1" XDG_STATE_HOME="$work/s1" XDG_CACHE_HOME="$work/c1" nvim -l "$repo/reference/lua/lua-uab-09-native.lua" "$source" "$vectors" 2>"$work/native.json";env XDG_DATA_HOME="$work/d2" XDG_STATE_HOME="$work/s2" XDG_CACHE_HOME="$work/c2" nvim -l "$repo/reference/lua/lua-uab-09-native.lua" "$work/projected.lua" "$vectors" 2>"$work/projected.json";node "$repo/reference/js/json-equal.mjs" "$work/native.json" "$work/projected.json";env XDG_DATA_HOME="$work/dd" XDG_STATE_HOME="$work/sd" XDG_CACHE_HOME="$work/cd" nvim -l "$repo/reference/lua/lua-uab-09-native.lua" "$source" "$vectors" denied 2>"$work/denied-native.json";rg -q '"denied":true' "$work/denied-native.json"
(cd "$repo/reference/go"&&go test -buildvcs=false ./canonicaleval ./wasmtarget&&go build -buildvcs=false -o "$work/eval" ./cmd/effect-eval&&go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower);"$work/eval" "$work/a.seme" "$vectors">"$work/canonical.json";"$work/lower" "$work/a.seme" "$work/a.wasm" "$work/abi.json";rg -q '"observability.log"' "$work/abi.json";node "$repo/reference/lua/lua-uab-09-wasm-runner.mjs" "$work/a.wasm" "$vectors">"$work/wasm.json";node -e 'const f=require("fs"),u=require("util"),a=JSON.parse(f.readFileSync(process.argv[1])),b=JSON.parse(f.readFileSync(process.argv[2])),c=JSON.parse(f.readFileSync(process.argv[3]));if(!u.isDeepStrictEqual(a.valid,b.valid)||!u.isDeepStrictEqual(a.valid,c.valid)||b.denied!=="no-events"||c.denied!=="no-events"||!u.isDeepStrictEqual(c.malformed,["empty","short","long","invalid-bool"]))process.exit(1)' "$work/native.json" "$work/canonical.json" "$work/wasm.json"
for x in capability argument;do node "$repo/reference/lua/lua-uab-09-forge.mjs" "$work/a.g1" "$work/$x.g1" "$x";"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/$x.g1" "$work/$x.seme";if "$work/lower" "$work/$x.seme" "$work/$x.wasm" "$work/$x.json">"$work/$x.log" 2>&1;then echo "accepted effect forgery $x" >&2;exit 1;fi;rg -q 'wasm\.' "$work/$x.log";done
mkdir "$work/pinned" "$work/pulp";git -C "$pulp_repo" archive "$commit"|tar -x -C "$work/pinned";cp "$repo/targets/wasm/pulp-effect-v1/capability.go" "$work/pinned/cmd/pulp-seme-function-proof/capability.go";(cd "$work/pinned"&&go build -buildvcs=false -o "$work/run" ./cmd/pulp-seme-function-proof);cp "$repo/targets/wasm/pulp-effect-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml";cp "$repo/targets/wasm/pulp-effect-v1/pulp.denied.cell.toml" "$work/pulp/pulp.denied.cell.toml";cp "$work/a.wasm" "$work/pulp/effect-function.wasm";node "$repo/reference/lua/lua-uab-09-wasm-runner.mjs" "$work/a.wasm" "$vectors" --tsv>"$work/q.tsv";:>"$work/ledger.tsv"
while IFS= read -r row; do
  kind=$(printf '%s\n' "$row" | cut -f1)
  name=$(printf '%s\n' "$row" | cut -f2)
  request=$(printf '%s\n' "$row" | cut -f3)
  response=$(printf '%s\n' "$row" | cut -f4)
  trace=$(printf '%s\n' "$row" | cut -f5)
  if [ "$kind" = valid ]; then
    "$work/run" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 -request "$request" >"$work/$name.log" 2>&1
    rg -q "\"response\":\"$response\"" "$work/$name.log"
    test "$(rg -c '^\[observability.log\]' "$work/$name.log")" -eq 2
    first=$(printf '%s' "$trace" | cut -d, -f1)
    second=$(printf '%s' "$trace" | cut -d, -f2)
    rg '^\[observability.log\]' "$work/$name.log" | sed -n '1p' | rg -q "value=$first"
    rg '^\[observability.log\]' "$work/$name.log" | sed -n '2p' | rg -q "value=$second"
    if "$work/run" -manifest "$work/pulp/pulp.denied.cell.toml" -provider seme.function-v1 -request "$request" >"$work/$name-denied.log" 2>&1; then
      echo "accepted absent capability for $name" >&2
      exit 1
    fi
    if rg -q '^\[observability.log\]' "$work/$name-denied.log"; then
      echo "observed an effect without its capability for $name" >&2
      exit 1
    fi
  else
    if "$work/run" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 -request "$request" >"$work/$name.log" 2>&1; then
      echo "accepted malformed request $name" >&2
      exit 1
    fi
    if rg -q '^\[observability.log\]' "$work/$name.log"; then
      echo "malformed request produced effects for $name" >&2
      exit 1
    fi
  fi
  printf '%s\n' "$row" >>"$work/ledger.tsv"
done <"$work/q.tsv"
cmp "$work/q.tsv" "$work/ledger.tsv"
sed 's/Seme.observe(first)/print(first)/' "$source">"$work/bad.lua";if node "$repo/reference/lua/lua-provider-cli.mjs" --source "$work/bad.lua" --module "$module" --package bad --revision 1 --entry Observe --out "$work/bad.g1" 2>"$work/bad.log";then exit 1;fi;rg -q 'lua.unsupported_statement:.*:[1-9][0-9]*:1' "$work/bad.log"
node -e 'const s=require(process.argv[1]),e=["lift","native_parity","target_parity","projection_round_trip","rejection"];require("assert").deepStrictEqual(s.languages.lua["UAB-09"],e)' "$repo/conformance/uab-v1/scorecard.json"
echo 'Lua UAB-09 complete acceptance: all five evidence classes pass'
