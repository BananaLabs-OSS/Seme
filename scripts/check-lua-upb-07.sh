#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-lua-upb-06.sh"
"$repo/scripts/check-lua-upb-07-foundation.sh"
"$repo/scripts/check-lua-upb-07-runtime.sh"
echo 'Lua UPB-07 all seven project evidence classes pass'
