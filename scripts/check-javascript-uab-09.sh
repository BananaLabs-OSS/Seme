#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd); pulp_repo=${PULP_REPO:-"$repo/../Pulp"}; commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/javascript-uab09.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
source="$repo/fixtures/javascript-uab-09/program.js"; vectors="$repo/fixtures/javascript-uab-09/vectors.json"; module="$repo/modules/execution/v20/module.g1"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$source" --module "$module" --package example.test/javascript-uab-09 --revision 1 --entry Observe --out "$work/source.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/source.g1" "$work/program.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/source.g1" "$work/projected.js"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.js" --module "$module" --package example.test/javascript-uab-09 --revision 1 --entry Observe --out "$work/relift.g1"
cmp "$work/source.g1" "$work/relift.g1"
node "$repo/reference/js/javascript-uab-09-native-runner.mjs" "$source" "$vectors" > "$work/original.json"
node "$repo/reference/js/javascript-uab-09-native-runner.mjs" "$work/projected.js" "$vectors" > "$work/projected.json"
(cd "$repo/reference/go" && go test -buildvcs=false ./canonicaleval ./wasmtarget && go build -buildvcs=false -o "$work/eval" ./cmd/effect-eval && go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
"$work/eval" "$work/program.seme" "$vectors" > "$work/canonical-all.json"
node -e 'let x=JSON.parse(require("fs").readFileSync(process.argv[1]));process.stdout.write(JSON.stringify({valid:x.valid}))' "$work/canonical-all.json" > "$work/canonical.json"
"$work/lower" "$work/program.seme" "$work/program.wasm" "$work/abi.json"
rg -q '"fidelity": "adapted"' "$work/abi.json"; rg -q '"observability.log"' "$work/abi.json"
node "$repo/reference/js/javascript-uab-09-wasm-runner.mjs" "$work/program.wasm" "$vectors" > "$work/wasm.json"
node "$repo/reference/js/json-equal.mjs" "$work/original.json" "$work/projected.json"
node "$repo/reference/js/json-equal.mjs" "$work/original.json" "$work/canonical.json"
node "$repo/reference/js/json-equal.mjs" "$work/original.json" "$work/wasm.json"
test -f "$pulp_repo/go.mod"; git -C "$pulp_repo" cat-file -e "$commit^{commit}"; mkdir "$work/pinned" "$work/pulp"
git -C "$pulp_repo" archive "$commit" | tar -x -C "$work/pinned"
cp "$repo/targets/wasm/pulp-effect-v1/capability.go" "$work/pinned/cmd/pulp-seme-function-proof/capability.go"
(cd "$work/pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-effect-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"; cp "$repo/targets/wasm/pulp-effect-v1/pulp.denied.cell.toml" "$work/pulp/pulp.denied.cell.toml"; cp "$work/program.wasm" "$work/pulp/effect-function.wasm"
node "$repo/reference/js/javascript-uab-09-wasm-runner.mjs" "$work/program.wasm" "$vectors" tsv > "$work/requests.tsv"; : > "$work/ledger.tsv"
tab=$(printf '\t'); while IFS="$tab" read -r name request response trace; do
  node "$repo/reference/js/pure-function-effect-runner.mjs" "$work/program.wasm" "$request" "$response" "$trace" denied > "$work/$name.wasm-denied.json"
  rg -q '"mode":"denied","events":\[\]' "$work/$name.wasm-denied.json"
  (cd "$work/pulp" && "$work/pulp-runner" -manifest pulp.cell.toml -provider seme.function-v1 -request "$request") > "$work/$name.log" 2>&1
  rg -q "\"response\":\"$response\"" "$work/$name.log"; test "$(rg -c '^\[observability.log\]' "$work/$name.log")" -eq 2
  first=$(printf '%s' "$trace" | cut -d, -f1); second=$(printf '%s' "$trace" | cut -d, -f2)
  rg '^\[observability.log\]' "$work/$name.log" | sed -n '1p' | rg -q "value=$first"; rg '^\[observability.log\]' "$work/$name.log" | sed -n '2p' | rg -q "value=$second"
  if (cd "$work/pulp" && "$work/pulp-runner" -manifest pulp.denied.cell.toml -provider seme.function-v1 -request "$request") > "$work/$name.denied" 2>&1; then exit 1; fi
  if rg -q '^\[observability.log\]' "$work/$name.denied"; then exit 1; fi
  printf '%s\t%s\t%s\t%s\n' "$name" "$request" "$response" "$trace" >> "$work/ledger.tsv"
done < "$work/requests.tsv"; cmp "$work/requests.tsv" "$work/ledger.tsv"
sed 's/console\.log(first)/console.error(first)/' "$source" > "$work/invalid.js"
if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/invalid.js" --module "$module" --package example.test/javascript-uab-09-invalid --revision 1 --entry Observe --out "$work/invalid.g1" > "$work/invalid.log" 2>&1; then exit 1; fi
rg -q 'javascript.unsupported_effect:[1-9][0-9]*:[1-9][0-9]*' "$work/invalid.log"
node -e 'const s=require(process.argv[1]),e=["lift","native_parity","target_parity","projection_round_trip","rejection"];require("assert").deepStrictEqual(s.languages.javascript["UAB-09"],e)' "$repo/conformance/uab-v1/scorecard.json"
echo 'JavaScript UAB-09 complete acceptance: all five evidence classes pass'
