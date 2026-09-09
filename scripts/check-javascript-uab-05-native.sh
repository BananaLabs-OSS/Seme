#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd); work=$(mktemp -d "${TMPDIR:-/tmp}/seme-js05.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
source="$repo/fixtures/javascript-uab-05/program.js"; vectors="$repo/fixtures/javascript-uab-05/vectors.json"; module="$repo/modules/execution/v27/module.g1"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$source" --module "$module" --package example.test/javascript-uab-05 --revision 1 --entry Dispatch --out "$work/source.g1"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/source.g1" "$work/projected.js"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.js" --module "$module" --package example.test/javascript-uab-05 --revision 1 --entry Dispatch --out "$work/relift.g1"
cmp "$work/source.g1" "$work/relift.g1"
node "$repo/reference/js/javascript-uab-05-native-runner.mjs" "$source" "$vectors" > "$work/original.json"
node "$repo/reference/js/javascript-uab-05-native-runner.mjs" "$work/projected.js" "$vectors" > "$work/projected.json"
node "$repo/reference/js/json-equal.mjs" "$work/original.json" "$work/projected.json"
sed '/export function Dispatch/i OffsetAdjuster.prototype.Adjust = function(value) { return value; };' "$source" > "$work/prototype-mutation.js"
if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/prototype-mutation.js" --module "$module" --package example.test/javascript-uab-05-reject --revision 1 --entry Dispatch --out "$work/bad.g1" 2> "$work/bad.log"; then echo 'prototype mutation accepted' >&2; exit 1; fi
rg -q 'javascript\.' "$work/bad.log"
echo 'JavaScript UAB-05 native candidate: structural witness lift, projection relift, both native witnesses, and prototype-mutation rejection pass; canonical/Wasm/Pulp unclaimed'
