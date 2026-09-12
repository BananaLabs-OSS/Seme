#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-lua-upb-05.sh"
"$repo/scripts/check-lua-upb-06-foundation.sh"
echo 'Lua UPB-06 all seven project evidence classes pass'
