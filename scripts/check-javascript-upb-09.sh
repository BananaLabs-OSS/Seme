#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.."&&pwd)
"$repo/scripts/check-javascript-upb-08.sh"
"$repo/scripts/check-javascript-upb-09-foundation.sh"
"$repo/scripts/check-javascript-upb-09-runtime.sh"
echo 'JavaScript UPB-09 cumulative authority gate passes'
