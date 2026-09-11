#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-javascript-upb-06.sh"
"$repo/scripts/check-javascript-upb-07-foundation.sh"
"$repo/scripts/check-javascript-upb-07-runtime.sh"
echo 'JavaScript UPB-07 cumulative authority gate passes'
