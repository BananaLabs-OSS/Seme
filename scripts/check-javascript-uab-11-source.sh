#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/javascript-uab11-source.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
package=seme.uab11/application

module="$repo/modules/execution/v35/module.g1"
cd "$repo"
node reference/js/javascript-package-provider-cli.mjs \
  --files fixtures/javascript-uab-11/application.js,fixtures/javascript-uab-11/policy.js \
  --module "$module" --package "$package" --revision 1 --entry Apply --out "$work/program.g1"
bootstrap/seme-k0-linux-amd64 compiler/g1-compiler.k0 "$work/program.g1" "$work/program.seme"
bootstrap/seme-k0-linux-amd64 compiler/kernel-wire-validator.k0 "$work/program.seme"
test "$(rg -c '^en [0-9a-f]{32} 0000000000000000000000000000a035 ' "$work/program.g1")" -eq 1
(cd reference/go && go build -buildvcs=false -o "$work/boundary" ./cmd/pure-application-boundary)
 (cd reference/go && go build -buildvcs=false -o "$work/canonical-observe" ./cmd/canonical-observe)
"$work/boundary" "$work/program.seme" > "$work/boundary.json"
node -e 'const b=require(process.argv[1]);if(b.parameters.length!==2||b.parameters[0].type!=="record"||b.parameters[1].type!=="record")throw Error("boundary_parameters");if(b.result.type!=="result<transition<record,i64>,i64>")throw Error("boundary_result");require("assert").deepStrictEqual(b.required_capabilities,["observability.log"])' "$work/boundary.json"

node reference/js/javascript-projector-cli.mjs "$work/program.g1" "$work/projected.mjs"
node --check "$work/projected.mjs"
node reference/js/javascript-provider-cli.mjs --source "$work/projected.mjs" --module "$module" \
  --package "$package" --revision 1 --entry Apply --out "$work/relift.g1"
cmp "$work/program.g1" "$work/relift.g1"
node reference/js/javascript-uab-11-native-runner.mjs "$repo/fixtures/javascript-uab-11/application.js" > "$work/original.json"
node reference/js/javascript-uab-11-native-runner.mjs "$work/projected.mjs" > "$work/projected.json"
cmp "$work/original.json" "$work/projected.json"
node reference/js/javascript-uab-11-canonical-corpus.mjs "$work/projected.mjs" "$work/requests.jsonl" "$work/expected.jsonl"
"$work/canonical-observe" "$work/program.seme" < "$work/requests.jsonl" > "$work/canonical.jsonl"
node reference/js/javascript-uab-11-jsonl-equal.mjs "$work/expected.jsonl" "$work/canonical.jsonl"

# Authorization is checked before evaluation; denial emits neither a value nor
# a partial effect trace.
sed -n '1p' "$work/requests.jsonl" | sed 's/"observability.log"/"denied"/' > "$work/denied.jsonl"
if "$work/canonical-observe" "$work/program.seme" < "$work/denied.jsonl" > "$work/denied.out" 2> "$work/denied.log"; then exit 1; fi
test ! -s "$work/denied.out"; rg -q 'canonicaleval.effect_denied' "$work/denied.log"

# A boundary value that lies about the command Boolean type rejects without an
# observation.
sed -n '1p' "$work/requests.jsonl" | sed 's/"Scale":{"kind":"bool","bool":true}/"Scale":{"kind":"i64","i64":"1"}/' > "$work/malformed-value.jsonl"
if "$work/canonical-observe" "$work/program.seme" < "$work/malformed-value.jsonl" > "$work/malformed-value.out" 2> "$work/malformed-value.log"; then exit 1; fi
test ! -s "$work/malformed-value.out"; rg -q 'canonicaleval' "$work/malformed-value.log"

# Exercise UAB-10 inside the cumulative fixture. Identity evidence is external
# to JavaScript; the renamed projection remains an ordinary native program.
node reference/js/javascript-uab-11-rename.mjs "$work/projected.mjs" "$work/renamed.mjs"
node reference/js/javascript-uab-10-evidence.mjs "$package" Apply Execute "$work/identity.json"
node reference/js/javascript-provider-cli.mjs --source "$work/renamed.mjs" --module "$module" \
  --package "$package" --revision 2 --entry Execute --identity-evidence "$work/identity.json" --out "$work/renamed.g1"
identity=$(node -e 'const x=require(process.argv[1]);process.stdout.write(x.renames[0].identity)' "$work/identity.json")
rg -q "^en $identity 00000000000000000000000000009011 " "$work/program.g1"
rg -q "^en $identity 00000000000000000000000000009011 " "$work/renamed.g1"
node reference/js/javascript-projector-cli.mjs "$work/renamed.g1" "$work/renamed-projected.mjs"
node reference/js/javascript-provider-cli.mjs --source "$work/renamed-projected.mjs" --module "$module" \
  --package "$package" --revision 2 --entry Execute --identity-evidence "$work/identity.json" --out "$work/renamed-relift.g1"
cmp "$work/renamed.g1" "$work/renamed-relift.g1"
node reference/js/javascript-uab-11-native-runner.mjs "$work/renamed-projected.mjs" > "$work/renamed.json"
cmp "$work/original.json" "$work/renamed.json"

if node reference/js/javascript-provider-cli.mjs --source fixtures/javascript-uab-11/reject-raw-index.js \
  --module "$module" --package seme.uab11/reject --revision 1 --entry RawIndex --out "$work/rejected.g1" > "$work/rejected.log" 2>&1; then exit 1; fi
rg -q 'javascript.raw_index_requires_adapter:[1-9][0-9]*:[1-9][0-9]*' "$work/rejected.log"
test ! -e "$work/rejected.g1"
for forgery in map-option-type transition-type effect-authority erase-stateful-call; do
  node reference/js/javascript-uab-11-forge.mjs "$work/program.g1" "$forgery" "$work/$forgery.g1"
  if node reference/js/javascript-projector-cli.mjs "$work/$forgery.g1" "$work/$forgery.mjs" > "$work/$forgery.log" 2>&1; then exit 1; fi
  rg -q 'javascript_projection\.' "$work/$forgery.log"
  test ! -e "$work/$forgery.mjs"
done
echo 'JavaScript UAB-11 source: lift, validation, native projection parity, identity rename, re-lift, and rejection pass'
