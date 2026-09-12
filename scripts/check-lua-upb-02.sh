#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-lua-upb-01.sh"
"$repo/scripts/check-lua-upb-02-graph.sh"
"$repo/scripts/check-lua-upb-02-parity.sh"
echo 'Lua UPB-02 all seven project evidence classes pass'
