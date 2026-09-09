#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-lua-compound-v1.sh"
"$repo/scripts/check-lua-composite-v32.sh"
node "$repo/reference/lua/lua-uab-02-evidence.mjs" "$repo/conformance/uab-v1/lua-uab-02-evidence.json"

# UAB-02 remains deliberately absent from the official scorecard until every
# family has all five required evidence categories, including target parity.
node -e '
  const report = require(process.argv[1]);
  if (report.languages.lua["UAB-02"].length !== 0) process.exit(1);
' "$repo/conformance/uab-v1/scorecard.json"

echo "Lua UAB-02 consolidated partial gate passed without claiming completion"
