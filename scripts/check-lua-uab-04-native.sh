#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-lua-uab04.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
module="$repo/modules/execution/v34/module.g1"
source="$repo/fixtures/lua-uab-04/program.lua"
composed="$repo/fixtures/lua-uab-04/composed.lua"
vectors="$repo/fixtures/lua-uab-04/vectors.json"
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
GOCACHE="$work/go-cache"
XDG_CACHE_HOME="$work/cache"
export GOCACHE XDG_CACHE_HOME
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/canonical-eval" ./cmd/canonical-eval && go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
for entry in ConstructSlice Append Update Remove Length Index Traverse EmptyMap Insert Lookup RemoveMap Evaluate; do
  entry_source=$source
  if [ "$entry" = Evaluate ]; then entry_source=$composed; fi
  node "$repo/reference/lua/lua-provider-cli.mjs" --source "$entry_source" --module "$module" --package "example.test/lua-uab04-$entry" --revision 1 --entry "$entry" --out "$work/$entry.g1"
  node "$repo/reference/lua/lua-projector-cli.mjs" "$work/$entry.g1" "$work/$entry.lua"
  node "$repo/reference/lua/lua-provider-cli.mjs" --source "$work/$entry.lua" --module "$module" --package "example.test/lua-uab04-$entry" --revision 1 --entry "$entry" --out "$work/$entry-relift.g1"
  cmp "$work/$entry.g1" "$work/$entry-relift.g1"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/$entry.g1" "$work/$entry.seme"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/$entry-relift.g1" "$work/$entry-relift.seme"
  cmp "$work/$entry.seme" "$work/$entry-relift.seme"
  node "$repo/reference/js/lua-uab-04-vector-adapter.mjs" canonical "$vectors" "$entry" > "$work/$entry-canonical-vectors.json"
  "$work/canonical-eval" "$work/$entry.seme" "$work/$entry-canonical-vectors.json" --named-rejections > "$work/$entry-canonical-raw.json"
  node "$repo/reference/js/lua-uab-04-vector-adapter.mjs" normalize "$vectors" "$entry" "$work/$entry-canonical-raw.json" > "$work/$entry-canonical.json"
  env XDG_DATA_HOME="$work/data-original-$entry" XDG_STATE_HOME="$work/state-original-$entry" XDG_CACHE_HOME="$work/cache-original-$entry" nvim -l "$repo/reference/lua/lua-uab-04-native.lua" "$entry_source" "$vectors" "$entry" > "$work/$entry-original.json"
  env XDG_DATA_HOME="$work/data-projected-$entry" XDG_STATE_HOME="$work/state-projected-$entry" XDG_CACHE_HOME="$work/cache-projected-$entry" nvim -l "$repo/reference/lua/lua-uab-04-native.lua" "$work/$entry.lua" "$vectors" "$entry" > "$work/$entry-projected.json"
  node -e 'const fs=require("fs"),assert=require("assert");assert.deepStrictEqual(JSON.parse(fs.readFileSync(process.argv[1])),JSON.parse(fs.readFileSync(process.argv[2])))' "$work/$entry-original.json" "$work/$entry-projected.json"
  node -e 'const fs=require("fs"),assert=require("assert");assert.deepStrictEqual(JSON.parse(fs.readFileSync(process.argv[1])),JSON.parse(fs.readFileSync(process.argv[2])))' "$work/$entry-original.json" "$work/$entry-canonical.json"
done

"$work/lower" "$work/Evaluate.seme" "$work/Evaluate.wasm" "$work/Evaluate-abi.json"
node "$repo/reference/js/lua-uab-04-wasm-runner.mjs" "$work/Evaluate.wasm" "$vectors" > "$work/Evaluate-wasm.json"
node -e 'const fs=require("fs"),assert=require("assert");for(const p of process.argv.slice(2))assert.deepStrictEqual(JSON.parse(fs.readFileSync(process.argv[1])),JSON.parse(fs.readFileSync(p)))' "$work/Evaluate-original.json" "$work/Evaluate-projected.json" "$work/Evaluate-canonical.json" "$work/Evaluate-wasm.json"
node "$repo/reference/js/lua-uab-04-wasm-runner.mjs" "$work/Evaluate.wasm" "$vectors" --tsv > "$work/Evaluate-requests.tsv"

test -f "$pulp_repo/go.mod"
git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/Evaluate.wasm" "$work/pulp/pure-function.wasm"
: > "$work/pulp-audit.tsv"
tab=$(printf '\t')
while IFS="$tab" read -r kind name request response; do
  if [ "$kind" = valid ]; then
    "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 -request "$request" > "$work/pulp-$name.log" 2>&1
    rg -q "\"response\":\"$response\"" "$work/pulp-$name.log"
    value=$(node -e 'const b=Buffer.from(process.argv[1],"hex");console.log(b.readBigInt64LE().toString())' "$response")
    printf 'valid\t%s\t%s\n' "$name" "$value" >> "$work/pulp-audit.tsv"
  else
    if "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 -request "$request" > "$work/pulp-$name.log" 2>&1; then
      echo "Pulp accepted $name" >&2
      exit 1
    fi
    printf 'malformed\t%s\trejected\n' "$name" >> "$work/pulp-audit.tsv"
  fi
done < "$work/Evaluate-requests.tsv"
node -e 'const fs=require("fs"),assert=require("assert"),want=JSON.parse(fs.readFileSync(process.argv[1])),rows=fs.readFileSync(process.argv[2],"utf8").trim().split("\n").map(x=>x.split("\t")),got={valid:{},malformed:{}};for(const [kind,name,value] of rows)kind==="valid"?got.valid[name]=value:got.malformed[name]=value==="rejected";assert.deepStrictEqual(got,want)' "$work/Evaluate-wasm.json" "$work/pulp-audit.tsv"
for mutation in raw-table raw-index wrong-index-kind nil-deletion raw-fold-arithmetic; do
  case "$mutation" in
    raw-table) entry=ConstructSlice; sed 's/return Seme.slice(first, second)/return {first, second}/' "$source" ;;
    raw-index) entry=Index; sed 's/return Seme.index_zero(values, index)/return values[index]/' "$source" ;;
    wrong-index-kind) entry=Index; sed 's/return Seme.index_zero(values, index)/return Seme.index_zero(values, "zero")/' "$source" ;;
    nil-deletion) entry=RemoveMap; sed 's/return Seme.map_remove(values, key)/values[key] = nil/' "$source" ;;
    raw-fold-arithmetic) entry=Traverse; sed 's/return Seme.add(accumulator, element)/return accumulator + element/' "$source" ;;
  esac > "$work/bad.lua"
  if cmp -s "$source" "$work/bad.lua"; then
    echo "Lua UAB-04 mutation did not alter source: $mutation" >&2
    exit 1
  fi
  if node "$repo/reference/lua/lua-provider-cli.mjs" --source "$work/bad.lua" --module "$module" --package example.test/lua-uab04-reject --revision 1 --entry "$entry" --out "$work/bad.g1" 2> "$work/bad.log"; then
    echo "Lua UAB-04 mutation accepted: $mutation" >&2
    exit 1
  fi
  rg -q 'lua\.' "$work/bad.log"
done
node -e 'const s=require(process.argv[1]),e=["lift","native_parity","target_parity","projection_round_trip","rejection"];if(JSON.stringify(s.languages.lua["UAB-04"])!==JSON.stringify(e))process.exit(1)' "$repo/conformance/uab-v1/scorecard.json"
echo "Lua UAB-04 complete evidence: native, projected, canonical, Wasm, and pinned Pulp observations agree"
