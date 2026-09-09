#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-lua-uab-11-source.sh"
"$repo/scripts/check-lua-uab-11-target.sh"
echo 'Lua UAB-11 complete acceptance: cumulative application and all five evidence classes pass'
