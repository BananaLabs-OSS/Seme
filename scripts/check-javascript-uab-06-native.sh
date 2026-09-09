#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd); work=$(mktemp -d "${TMPDIR:-/tmp}/seme-js06.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
source="$repo/fixtures/javascript-uab-06/program.js"; vectors="$repo/fixtures/javascript-uab-06/vectors.json"; module="$repo/modules/execution/v29/module.g1"
node "$repo/reference/js/javascript-uab-06-native-runner.mjs" "$source" "$vectors" > "$work/original.json"
for pair in Immutable:immutable Mutable:mutable; do
  entry=${pair%%:*}; family=${pair#*:}
  node "$repo/reference/js/javascript-provider-cli.mjs" --source "$source" --module "$module" --package example.test/javascript-uab-06 --revision 1 --entry "$entry" --out "$work/$family.g1"
  node "$repo/reference/js/javascript-projector-cli.mjs" "$work/$family.g1" "$work/$family.js"
  node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/$family.js" --module "$module" --package example.test/javascript-uab-06 --revision 1 --entry "$entry" --out "$work/$family-relift.g1"
  cmp "$work/$family.g1" "$work/$family-relift.g1"
  node "$repo/reference/js/javascript-uab-06-native-runner.mjs" "$work/$family.js" "$vectors" "$family" > "$work/$family.json"
done
node -e 'const fs=require("fs"),u=require("util"),a=JSON.parse(fs.readFileSync(process.argv[1])).valid,b={...JSON.parse(fs.readFileSync(process.argv[2])).valid,...JSON.parse(fs.readFileSync(process.argv[3])).valid};if(!u.isDeepStrictEqual(a,b))process.exit(1)' "$work/original.json" "$work/immutable.json" "$work/mutable.json"
sed 's/return (value) =>/return function(value) { return this.base + value; }; \/\//' "$source" > "$work/dynamic-this.js"
if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/dynamic-this.js" --module "$module" --package example.test/javascript-uab-06-reject --revision 1 --entry Immutable --out "$work/bad.g1" 2> "$work/bad.log"; then echo 'dynamic this accepted' >&2; exit 1; fi
rg -q 'javascript\.' "$work/bad.log"
echo 'JavaScript UAB-06 native candidate: one source, immutable/mutable capture outcomes, projection relift, and dynamic-this rejection pass; canonical/Wasm/Pulp unclaimed'
