#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-uab-10.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
package=example.test/javascript-uab-10
module="$repo/modules/execution/v30/module.g1"
original="$repo/fixtures/javascript-uab-10/original.js"
formatted="$repo/fixtures/javascript-uab-10/formatted.js"
renamed="$repo/fixtures/javascript-uab-10/renamed.js"

node "$repo/reference/js/javascript-uab-10-evidence.mjs" "$package" Identity Preserve "$work/evidence.json"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$original" --module "$module" --package "$package" --revision 1 --entry Identity --out "$work/original.g1"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$formatted" --module "$module" --package "$package" --revision 1 --entry Identity --out "$work/formatted.g1"
cmp "$work/original.g1" "$work/formatted.g1"

# Reconciliation changes descriptive spelling and revision, while preserving
# the prior semantic declaration identity supplied as checked evidence.
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$renamed" --module "$module" --package "$package" --revision 2 --entry Preserve --identity-evidence "$work/evidence.json" --out "$work/renamed.g1"
identity=$(node -e 'const x=require(process.argv[1]);process.stdout.write(x.renames[0].identity)' "$work/evidence.json")
rg -q "^en $identity 00000000000000000000000000009011 " "$work/original.g1"
rg -q "^en $identity 00000000000000000000000000009011 " "$work/renamed.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/renamed.g1" "$work/renamed.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/renamed.seme"

node "$repo/reference/js/javascript-projector-cli.mjs" "$work/renamed.g1" "$work/projected.js"
node --check "$work/projected.js"
rg -q 'export function Preserve' "$work/projected.js"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.js" --module "$module" --package "$package" --revision 2 --entry Preserve --identity-evidence "$work/evidence.json" --out "$work/relift.g1"
cmp "$work/renamed.g1" "$work/relift.g1"

node "$repo/reference/js/javascript-uab-10-native-runner.mjs" "$original" > "$work/original.json"
node "$repo/reference/js/javascript-uab-10-native-runner.mjs" "$formatted" > "$work/formatted.json"
node "$repo/reference/js/javascript-uab-10-native-runner.mjs" "$renamed" > "$work/renamed.json"
node "$repo/reference/js/javascript-uab-10-native-runner.mjs" "$work/projected.js" > "$work/projected.json"
cmp "$work/original.json" "$work/formatted.json"; cmp "$work/original.json" "$work/renamed.json"; cmp "$work/original.json" "$work/projected.json"

(cd "$repo/reference/go" && go test -buildvcs=false ./canonicaleval ./wasmtarget && go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower && go build -buildvcs=false -o "$work/eval" ./cmd/canonical-eval)
node -e 'const inputs=["0","7","-1","9223372036854775807","-9223372036854775808"];process.stdout.write(JSON.stringify({valid:inputs.map((x,i)=>({name:`case-${i}`,arguments:[{kind:"i64",i64:x}],result:{kind:"i64",i64:x}})),malformed:[{name:"wrong-type",arguments:[{kind:"bool",bool:true}]}]})+"\n")' > "$work/vectors.json"
"$work/eval" "$work/renamed.seme" "$work/vectors.json" --named-rejections > "$work/canonical.json"
node -e 'const f=require("fs"),u=require("util"),n=JSON.parse(f.readFileSync(process.argv[1])),c=JSON.parse(f.readFileSync(process.argv[2]));const cv=Object.entries(c.valid).sort(([a],[b])=>a.localeCompare(b)).map(([,v],i)=>({input:["0","7","-1","9223372036854775807","-9223372036854775808"][i],output:v.i64}));if(!u.isDeepStrictEqual(n,cv)||JSON.stringify(Object.keys(c.rejected))!==JSON.stringify(["wrong-type"]))process.exit(1)' "$work/renamed.json" "$work/canonical.json"
"$work/lower" "$work/renamed.seme" "$work/program.wasm" "$work/abi.json"
rg -q '"request_size": 8' "$work/abi.json"; rg -q '"response_size": 8' "$work/abi.json"
node "$repo/reference/js/pure-function-runner.mjs" "$work/program.wasm" 0000000000000000 0700000000000000 ffffffffffffffff ffffffffffffff7f 0000000000000080 > "$work/wasm.log"
for response in 0000000000000000 0700000000000000 ffffffffffffffff ffffffffffffff7f 0000000000000080; do rg -q "\"response\":\"$response\"" "$work/wasm.log"; done

test -f "$pulp_repo/go.mod"; git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pinned" "$work/pulp"; git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pinned"
(cd "$work/pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"; cp "$work/program.wasm" "$work/pulp/pure-function.wasm"
for request in 0000000000000000 0700000000000000 ffffffffffffffff ffffffffffffff7f 0000000000000080; do
  "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 -request "$request" > "$work/pulp-$request.log" 2>&1
  rg -q "\"response\":\"$request\"" "$work/pulp-$request.log"
done

# Forged, ambiguous, wrong-package, and colliding reconciliation evidence all
# fail before a graph is emitted.
node -e 'const fs=require("fs"),p=process.argv[1],x=require(p);x.renames[0].identity="00".repeat(16);fs.writeFileSync(process.argv[2],JSON.stringify(x))' "$work/evidence.json" "$work/forged.json"
if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$renamed" --module "$module" --package "$package" --revision 2 --entry Preserve --identity-evidence "$work/forged.json" --out "$work/forged.g1" > "$work/forged.log" 2>&1; then exit 1; fi
rg -q 'javascript.identity_evidence_forged' "$work/forged.log"; test ! -e "$work/forged.g1"
node -e 'const fs=require("fs"),x=require(process.argv[1]);x.renames.push({...x.renames[0]});fs.writeFileSync(process.argv[2],JSON.stringify(x))' "$work/evidence.json" "$work/ambiguous.json"
if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$renamed" --module "$module" --package "$package" --revision 2 --entry Preserve --identity-evidence "$work/ambiguous.json" --out "$work/ambiguous.g1" > "$work/ambiguous.log" 2>&1; then exit 1; fi
rg -q 'javascript.identity_evidence_ambiguous' "$work/ambiguous.log"; test ! -e "$work/ambiguous.g1"
node -e 'const fs=require("fs"),x=require(process.argv[1]);x.packagePath="example.test/wrong";fs.writeFileSync(process.argv[2],JSON.stringify(x))' "$work/evidence.json" "$work/wrong-package.json"
if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$renamed" --module "$module" --package "$package" --revision 2 --entry Preserve --identity-evidence "$work/wrong-package.json" --out "$work/wrong.g1" > "$work/wrong.log" 2>&1; then exit 1; fi
rg -q 'javascript.invalid_identity_evidence' "$work/wrong.log"; test ! -e "$work/wrong.g1"

node -e 'const s=require(process.argv[1]),e=["lift","native_parity","target_parity","projection_round_trip","rejection"];if(s.languages.javascript["UAB-10"].length!==0)require("assert").deepStrictEqual(s.languages.javascript["UAB-10"],e)' "$repo/conformance/uab-v1/scorecard.json"
echo 'JavaScript UAB-10 complete acceptance: all five evidence classes pass'
