#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/lua-uab-11-native.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
node "$repo/reference/lua/lua-uab-11-corpus.mjs" >"$work/corpus.json"
env XDG_DATA_HOME="$work/data" XDG_STATE_HOME="$work/state" XDG_CACHE_HOME="$work/cache" \
  nvim -l "$repo/reference/lua/lua-uab-11-native.lua" \
  "$repo/reference/lua/seme-values.lua" "$repo/fixtures/lua-uab-11/policy.lua" "$repo/fixtures/lua-uab-11/application.lua" "$work/corpus.json" >"$work/ledger.json"
node -e 'const x=require(process.argv[1]);if(x.observations.length!==2048||x.success+x.missing+x.index+x.negative!==2048||x.traces!==x.success||x.observations.filter(v=>v.outcome==="ok"&&v.trace.length!==1).length||x.observations.filter(v=>v.outcome==="error"&&v.trace.length!==0).length)process.exit(1)' "$work/ledger.json"
echo 'Lua UAB-11 native scaffold: fixed cases and 128 deterministic 16-command sequences pass'
