#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-lua-upb-07.sh"
"$repo/scripts/check-lua-upb-08-foundation.sh"
"$repo/scripts/check-lua-upb-08-runtime.sh"
echo 'Lua UPB-08 all eight project evidence classes pass'
