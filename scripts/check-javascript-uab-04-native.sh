#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
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
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/canonical-eval" ./cmd/canonical-eval)
"$work/canonical-eval" "$work/program.seme" "$work/canonical-vectors.json" > "$work/canonical.json"
node "$repo/reference/js/javascript-uab-04-compare.mjs" "$work/expected.json" "$work/canonical.json"

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
sed 's/BigInt.asIntN(64, total + Seme.index(revised, cursor))/total + Seme.index(revised, cursor)/' "$source" > "$work/raw-arithmetic.js"
reject raw-arithmetic 'javascript.i64_arithmetic_requires_asIntN'
sed 's/Seme.mapLookupZero/Seme.rawMapLookup/' "$source" > "$work/raw-missing.js"
reject raw-missing 'javascript.seme_constructor_type'

echo 'JavaScript UAB-04 scaffold: original/projected/canonical exact observations, immutable adapters, relift, validation, and direct rejection matrix pass; Wasm/Pulp remain unclaimed'
