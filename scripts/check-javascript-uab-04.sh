#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-javascript-uab-04-native.sh"
node -e 'const s=require(process.argv[1]);const e=["lift","native_parity","target_parity","projection_round_trip","rejection"];if(JSON.stringify(s.languages.javascript["UAB-04"])!==JSON.stringify(e))process.exit(1)' "$repo/conformance/uab-v1/scorecard.json"
echo 'JavaScript UAB-04 complete acceptance: all five evidence classes pass'
