#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/javascript-uab11-native.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
node "$repo/reference/js/javascript-uab-11-native-runner.mjs" "$repo/fixtures/javascript-uab-11/application.js" > "$work/first.json"
node "$repo/reference/js/javascript-uab-11-native-runner.mjs" "$repo/fixtures/javascript-uab-11/application.js" > "$work/second.json"
cmp "$work/first.json" "$work/second.json"
node -e 'const x=require(process.argv[1]);if(x.length!==128)throw Error("sequence_count");if(x.some(s=>s.steps.length!==16))throw Error("command_count");const all=x.flatMap(s=>s.steps);for(const code of ["1","2","3"])if(!all.some(v=>v.outcome.tag==="error"&&v.outcome.value===code))throw Error(`missing_error_${code}`);if(!all.some(v=>v.outcome.tag==="ok"))throw Error("missing_success")' "$work/first.json"
echo 'JavaScript UAB-11 native oracle: deterministic 128x16 cumulative sequences pass'
