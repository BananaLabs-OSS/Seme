#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.."&&pwd)
"$repo/scripts/check-lua-upb-08.sh"
"$repo/scripts/check-lua-upb-09-foundation.sh"
"$repo/scripts/check-lua-upb-09-runtime.sh"
echo 'Lua UPB-09 all nine project evidence classes pass'
