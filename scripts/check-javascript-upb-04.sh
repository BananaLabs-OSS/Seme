#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-javascript-upb-03.sh"
JS_UPB_FIXTURE="$repo/fixtures/javascript-upb04-typed" JS_UPB_PROJECT=example.test/javascript-upb04 \
JS_UPB_FILES=application.js,policy/collections.js JS_UPB_LOCAL_FILE=policy/collections.js \
JS_UPB_LOCAL_IDENTITY=example.test/javascript-upb04/policy/collections JS_UPB_LOCAL_FROM=example.test/javascript-upb04/application \
  "$repo/scripts/check-javascript-upb-03-dependency.sh"
"$repo/scripts/check-javascript-upb-04-parity.sh"
echo 'JavaScript UPB-04: seven independent evidence classes pass'
