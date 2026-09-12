#!/bin/sh
# Sole cumulative Lua UPB-11 transactional reconciliation authority gate.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.."&&pwd);work=$(mktemp -d "${TMPDIR:-/tmp}/seme-lua-upb11.XXXXXX");trap 'rm -rf "$work"' EXIT HUP INT TERM;GOCACHE="$work/go-cache";XDG_CACHE_HOME="$work/cache";export GOCACHE XDG_CACHE_HOME
LUA_UPB10_EXPORT="$work/upb10" "$repo/scripts/check-lua-upb-10.sh"
"$repo/scripts/check-language-service-v1.sh";"$repo/scripts/check-project-contract-v14.sh"
node --test "$repo/reference/lua/lua-provider.test.mjs" "$repo/reference/lua/lua-project-graph.test.mjs" "$repo/reference/lua/lua-project-reconcile.test.mjs"
(cd "$repo/reference/go"&&go test -p=1 -count=1 ./goupb09bundle ./patchinstance ./projectv14instance ./cmd/project-v12-bundle ./cmd/project-v14-finalize)
"$repo/scripts/check-lua-upb-11-reconciliation.sh"
LUA_UPB10_INPUT="$work/upb10" "$repo/scripts/check-lua-upb-11-project.sh"
echo 'Lua UPB-11 seven-class transactional project reconciliation authority gate passes'
