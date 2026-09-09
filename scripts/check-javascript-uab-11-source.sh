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

node reference/js/javascript-projector-cli.mjs "$work/program.g1" "$work/projected.mjs"
node --check "$work/projected.mjs"
node reference/js/javascript-provider-cli.mjs --source "$work/projected.mjs" --module "$module" \
  --package "$package" --revision 1 --entry Apply --out "$work/relift.g1"
cmp "$work/program.g1" "$work/relift.g1"
node reference/js/javascript-uab-11-native-runner.mjs "$repo/fixtures/javascript-uab-11/application.js" > "$work/original.json"
node reference/js/javascript-uab-11-native-runner.mjs "$work/projected.mjs" > "$work/projected.json"
cmp "$work/original.json" "$work/projected.json"

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
for forgery in map-option-type transition-type effect-authority; do
  node reference/js/javascript-uab-11-forge.mjs "$work/program.g1" "$forgery" "$work/$forgery.g1"
  if node reference/js/javascript-projector-cli.mjs "$work/$forgery.g1" "$work/$forgery.mjs" > "$work/$forgery.log" 2>&1; then exit 1; fi
  rg -q 'javascript_projection\.' "$work/$forgery.log"
  test ! -e "$work/$forgery.mjs"
done
echo 'JavaScript UAB-11 source: lift, validation, native projection parity, identity rename, re-lift, and rejection pass'
