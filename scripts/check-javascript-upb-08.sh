#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-javascript-upb-07.sh"
"$repo/scripts/check-javascript-upb-08-foundation.sh"
"$repo/scripts/check-javascript-upb-08-runtime.sh"
echo 'JavaScript UPB-08 cumulative authority gate passes'
