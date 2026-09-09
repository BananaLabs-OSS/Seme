#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}; pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-uab04.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
source="$repo/fixtures/javascript-uab-04/program.js"
vectors="$repo/fixtures/javascript-uab-04/vectors.json"
module="$repo/modules/execution/v34/module.g1"

node "$repo/reference/js/javascript-provider-cli.mjs" --source "$source" --module "$module" --package example.test/javascript-uab-04 --revision 1 --entry Evaluate --out "$work/source.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/source.g1" "$work/program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/program.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/source.g1" "$work/projected.js"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.js" --module "$module" --package example.test/javascript-uab-04 --revision 1 --entry Evaluate --out "$work/relift.g1"
cmp "$work/source.g1" "$work/relift.g1"
node "$repo/reference/js/javascript-uab-04-native-runner.mjs" "$source" "$vectors" > "$work/original.json"
node "$repo/reference/js/javascript-uab-04-native-runner.mjs" "$work/projected.js" "$vectors" > "$work/projected.json"
node "$repo/reference/js/json-equal.mjs" "$work/original.json" "$work/projected.json"
node "$repo/reference/js/javascript-uab-04-vectors.mjs" expected "$vectors" > "$work/expected.json"
node "$repo/reference/js/javascript-uab-04-compare.mjs" "$work/expected.json" "$work/original.json" "$work/projected.json"
node "$repo/reference/js/javascript-uab-04-vectors.mjs" canonical "$vectors" > "$work/canonical-vectors.json"
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/canonical-eval" ./cmd/canonical-eval && go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
"$work/canonical-eval" "$work/program.seme" "$work/canonical-vectors.json" > "$work/canonical.json"
node "$repo/reference/js/javascript-uab-04-compare.mjs" "$work/expected.json" "$work/canonical.json"
"$work/lower" "$work/program.seme" "$work/program.wasm" "$work/abi.json"
node "$repo/reference/js/javascript-uab-04-wasm-runner.mjs" "$work/program.wasm" "$vectors" > "$work/wasm.json"
node "$repo/reference/js/javascript-uab-04-compare.mjs" "$work/expected.json" "$work/wasm.json"
node "$repo/reference/js/javascript-uab-04-wasm-runner.mjs" "$work/program.wasm" "$vectors" --tsv > "$work/requests.tsv"

test -f "$pulp_repo/go.mod"; git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pulp" "$work/pulp-pinned"; git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp/pulp.cell.toml"; cp "$work/program.wasm" "$work/pulp/pure-function.wasm"; : > "$work/pulp-audit.tsv"
tab=$(printf '\t')
while IFS="$tab" read -r kind name request response; do
  if [ "$kind" = valid ]; then
    "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 -request "$request" > "$work/pulp-$name.log" 2>&1
    rg -q "\"response\":\"$response\"" "$work/pulp-$name.log"
    value=$(node -e 'const b=Buffer.from(process.argv[1],"hex");console.log(b.readBigInt64LE().toString())' "$response")
    printf 'valid\t%s\t%s\n' "$name" "$value" >> "$work/pulp-audit.tsv"
  else
    if "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 -request "$request" > "$work/pulp-$name.log" 2>&1; then echo "Pulp accepted $name" >&2; exit 1; fi
    printf 'malformed\t%s\trejected\n' "$name" >> "$work/pulp-audit.tsv"
  fi
done < "$work/requests.tsv"
node "$repo/reference/js/javascript-uab-04-pulp-audit.mjs" "$vectors" "$work/pulp-audit.tsv"

reject() {
  name=$1
  pattern=$2
  if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/$name.js" --module "$module" --package "example.test/javascript-uab-04-reject-$name" --revision 1 --entry Evaluate --out "$work/$name.g1" 2> "$work/$name.log"; then
    echo "JavaScript UAB-04 accepted $name" >&2; exit 1
  fi
  rg -q "$pattern" "$work/$name.log"
}
sed 's/]), index, replacement)/]), Number(index), replacement)/' "$source" > "$work/index-coercion.js"
reject index-coercion 'javascript.unsupported_call|javascript.unsupported_expression'
sed '0,/Seme.mapRemove/s//Seme.rawMapRemove/' "$source" > "$work/removal-operation.js"
reject removal-operation 'javascript.seme_constructor_type'
sed 's/BigInt.asIntN(64, sum + value)/sum + value/' "$source" > "$work/raw-arithmetic.js"
reject raw-arithmetic 'javascript.i64_arithmetic_requires_asIntN'
sed 's/Seme.mapLookupZero/Seme.rawMapLookup/' "$source" > "$work/raw-missing.js"
reject raw-missing 'javascript.seme_constructor_type'

echo 'JavaScript UAB-04 complete evidence candidate: original/projected/canonical/Wasm/Pulp exact observations and direct rejection pass'
