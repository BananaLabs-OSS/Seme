#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

(cd "$repo/reference/js" && npm test)
"$repo/scripts/check-javascript-uab-02-scalars.sh"
"$repo/scripts/check-javascript-uab-02-collections.sh"
"$repo/scripts/check-javascript-composite-v32.sh"

node -e '
  const report = require(process.argv[1]);
  const expected = ["lift", "native_parity", "target_parity", "projection_round_trip", "rejection"];
  if (JSON.stringify(report.languages.javascript["UAB-02"]) !== JSON.stringify(expected)) process.exit(1);
' "$repo/conformance/uab-v1/scorecard.json"

echo "JavaScript UAB-02 complete acceptance: original/projected native, canonical evaluator, Wasm, Pulp, and rejection evidence pass"
