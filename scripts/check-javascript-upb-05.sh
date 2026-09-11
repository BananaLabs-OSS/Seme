#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-javascript-upb-04.sh"
"$repo/scripts/check-javascript-upb-05-foundation.sh"
"$repo/scripts/check-javascript-upb-05-parity.sh"
echo 'JavaScript UPB-05: seven independent evidence classes pass'
