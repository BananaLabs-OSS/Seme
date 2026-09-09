#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

"$repo/scripts/check-javascript-uab-11-native.sh"
"$repo/scripts/check-javascript-uab-11-source.sh"
"$repo/scripts/check-javascript-uab-11-target.sh"

# The three subordinate gates jointly and directly establish the frozen five
# evidence classes: source proves lift/projection/re-lift/rejection, native plus
# canonical proves behavioral parity, and the target gate proves Wasm/Pulp
# parity as well as atomic target rejection.
echo 'JavaScript UAB-11 complete acceptance: lift, native_parity, target_parity, projection_round_trip, and rejection pass'
