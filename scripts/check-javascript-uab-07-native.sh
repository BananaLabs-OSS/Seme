#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd); work=$(mktemp -d "${TMPDIR:-/tmp}/seme-js07.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
source="$repo/fixtures/javascript-uab-07/program.js"; vectors="$repo/fixtures/javascript-uab-07/vectors.json"; module="$repo/modules/execution/v27/module.g1"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$source" --module "$module" --package example.test/javascript-uab-07 --revision 1 --entry Step --out "$work/source.g1"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/source.g1" "$work/projected.js"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.js" --module "$module" --package example.test/javascript-uab-07 --revision 1 --entry Step --out "$work/relift.g1"; cmp "$work/source.g1" "$work/relift.g1"
node "$repo/reference/js/javascript-uab-07-native-runner.mjs" "$source" "$vectors" > "$work/native.json"; node "$repo/reference/js/javascript-uab-07-native-runner.mjs" "$work/projected.js" "$vectors" > "$work/projected.json"; node "$repo/reference/js/json-equal.mjs" "$work/native.json" "$work/projected.json"
sed 's/result: counter.Value/result: state.Value/' "$source" > "$work/alias.js"; if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/alias.js" --module "$module" --package bad --revision 1 --entry Step --out "$work/bad.g1" 2> "$work/bad.log"; then echo 'undeclared alias accepted' >&2; exit 1; fi; rg -q 'javascript\.' "$work/bad.log"
echo 'JavaScript UAB-07 native candidate: immutable state/result distinction, overflow, projection relift, and alias rejection pass'
