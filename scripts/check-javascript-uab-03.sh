#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}; pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-uab03.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
source="$repo/fixtures/javascript-uab-03/program.js"; vectors="$work/vectors.json"
node "$repo/reference/js/javascript-uab-03-vectors.mjs" native > "$vectors"
node "$repo/reference/js/javascript-uab-03-vectors.mjs" canonical > "$work/canonical-vectors.json"

node "$repo/reference/js/javascript-provider-cli.mjs" --source "$source" --module "$repo/modules/execution/v32/module.g1" --package example.test/javascript-uab-03 --revision 1 --out "$work/source.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/source.g1" "$work/program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/program.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/source.g1" "$work/projected.js"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.js" --module "$repo/modules/execution/v32/module.g1" --package example.test/javascript-uab-03 --revision 1 --out "$work/relifted.g1"
cmp "$work/source.g1" "$work/relifted.g1"
node "$repo/reference/js/javascript-uab-03-runner.mjs" "$source" "$vectors" > "$work/native.json"
node "$repo/reference/js/javascript-uab-03-runner.mjs" "$work/projected.js" "$vectors" > "$work/projected-native.json"
node "$repo/reference/js/json-equal.mjs" "$work/native.json" "$work/projected-native.json"

(cd "$repo/reference/go" && go test -buildvcs=false ./canonicaleval ./wasmtarget && go build -buildvcs=false -o "$work/canonical-eval" ./cmd/canonical-eval && go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
"$work/canonical-eval" "$work/program.seme" "$work/canonical-vectors.json" > "$work/canonical.json"
node -e 'const fs=require("fs"),u=require("util"),a=JSON.parse(fs.readFileSync(process.argv[1])).valid,x=JSON.parse(fs.readFileSync(process.argv[2])).valid,b={}; for(const [k,v] of Object.entries(x))b[k]=v.i64; if(!u.isDeepStrictEqual(a,b))process.exit(1)' "$work/native.json" "$work/canonical.json"
"$work/lower" "$work/program.seme" "$work/program.wasm" "$work/abi.json"
node "$repo/reference/js/javascript-uab-03-wasm-runner.mjs" "$work/program.wasm" "$vectors" > "$work/wasm.json"
node -e 'const fs=require("fs"),u=require("util"),a=JSON.parse(fs.readFileSync(process.argv[1])).valid,b=JSON.parse(fs.readFileSync(process.argv[2])).valid; if(!u.isDeepStrictEqual(a,b))process.exit(1)' "$work/native.json" "$work/wasm.json"

test -f "$pulp_repo/go.mod"; git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pulp" "$work/pulp-pinned"; git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp/pulp.cell.toml"; cp "$work/program.wasm" "$work/pulp/pure-function.wasm"
node "$repo/reference/js/javascript-uab-03-wasm-runner.mjs" "$work/program.wasm" "$vectors" --tsv > "$work/requests.tsv"
tab=$(printf '\t')
while IFS="$tab" read -r kind name request response; do
  if [ "$kind" = valid ]; then
    (cd "$work/pulp" && "$work/pulp-runner" -manifest pulp.cell.toml -provider seme.function-v2 -request "$request") > "$work/pulp-$name.log" 2>&1
    rg -q "\"response\":\"$response\"" "$work/pulp-$name.log"
  elif (cd "$work/pulp" && "$work/pulp-runner" -manifest pulp.cell.toml -provider seme.function-v2 -request "$request") > "$work/pulp-malformed-$name.log" 2>&1; then
    echo "Pulp accepted malformed UAB-03 request $name" >&2; exit 1
  fi
done < "$work/requests.tsv"

sed 's/BigInt.asIntN(64, result + 1n)/result + 1n/' "$source" > "$work/incompatible.js"
if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/incompatible.js" --module "$repo/modules/execution/v32/module.g1" --package example.test/javascript-uab-03-reject --revision 1 --out "$work/incompatible.g1" 2> "$work/rejection.log"; then
  echo "JavaScript Number/i64 conflation accepted" >&2; exit 1
fi
rg -q 'javascript.i64_arithmetic_requires_asIntN:18:' "$work/rejection.log"
node -e 'const r=require(process.argv[1]),e=["lift","native_parity","target_parity","projection_round_trip","rejection"];if(JSON.stringify(r.languages.javascript["UAB-03"])!==JSON.stringify(e))process.exit(1)' "$repo/conformance/uab-v1/scorecard.json"
echo "JavaScript UAB-03 complete acceptance: all five evidence classes pass"
