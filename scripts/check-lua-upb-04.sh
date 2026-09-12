#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-lua-upb-03.sh"
LUA_UPB_FIXTURE="$repo/fixtures/lua-upb04-typed" LUA_UPB_PROJECT=example.test/lua-upb04 LUA_UPB_FILES=application.lua,policy/collections.lua LUA_UPB_LOCAL_FILE=policy/collections.lua LUA_UPB_LOCAL_IDENTITY=example.test/lua-upb04/policy/collections LUA_UPB_LOCAL_FROM=example.test/lua-upb04/application LUA_UPB_ROCKSPEC=seme-lua-upb04-1.0-1.rockspec "$repo/scripts/check-lua-upb-03-dependency.sh"
"$repo/scripts/check-lua-upb-04-parity.sh"
echo 'Lua UPB-04 all seven project evidence classes pass'
