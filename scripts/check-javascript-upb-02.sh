#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-javascript-upb-01.sh"
"$repo/scripts/check-javascript-upb-02-graph.sh"
"$repo/scripts/check-javascript-upb-02-parity.sh"
echo 'JavaScript UPB-02: seven independent evidence classes pass'
