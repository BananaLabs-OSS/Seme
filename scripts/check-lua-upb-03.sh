#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-lua-upb-02.sh"
"$repo/scripts/check-lua-upb-03-dependency.sh"
LUA_UPB_FIXTURE="$repo/fixtures/lua-upb03-dependency" LUA_UPB_PACKAGE=example.test/lua-upb03 "$repo/scripts/check-lua-upb-02-parity.sh"
echo 'Lua UPB-03 all seven project evidence classes pass'
