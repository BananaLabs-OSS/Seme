#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-javascript-upb-02.sh"
"$repo/scripts/check-javascript-upb-03-dependency.sh"
JS_UPB_FIXTURE="$repo/fixtures/javascript-upb03-dependency" JS_UPB_PACKAGE=example.test/javascript-upb03 \
  "$repo/scripts/check-javascript-upb-02-parity.sh"
echo 'JavaScript UPB-03: seven independent evidence classes pass'
