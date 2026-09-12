#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-lua-upb-04.sh"
"$repo/scripts/check-lua-upb-05-foundation.sh"
"$repo/scripts/check-lua-upb-05-parity.sh"
echo 'Lua UPB-05 all seven project evidence classes pass'
