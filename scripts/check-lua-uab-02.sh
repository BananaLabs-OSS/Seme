#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-lua-compound-v1.sh"
"$repo/scripts/check-lua-composite-v32.sh"
"$repo/scripts/check-lua-uab-02-scalars.sh"
"$repo/scripts/check-lua-uab-02-aggregate.sh"
node "$repo/reference/lua/lua-uab-02-evidence.mjs" "$repo/conformance/uab-v1/lua-uab-02-evidence.json"

node -e '
  const report = require(process.argv[1]);
  const expected = ["lift", "native_parity", "target_parity", "projection_round_trip", "rejection"];
  if (JSON.stringify(report.languages.lua["UAB-02"]) !== JSON.stringify(expected)) process.exit(1);
' "$repo/conformance/uab-v1/scorecard.json"

echo "Lua UAB-02 complete acceptance: original/projected native, canonical evaluator, Wasm, Pulp, and rejection evidence pass"
