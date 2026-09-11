#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-javascript-upb-05.sh"
"$repo/scripts/check-javascript-upb-06-foundation.sh"
echo 'JavaScript UPB-06: seven independent evidence classes pass'
