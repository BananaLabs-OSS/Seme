#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-upb-v1-claimed.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

"$repo/scripts/check-upb-v1-scorecard.sh"
node --input-type=module - "$repo/conformance/upb-v1/evidence-gates.json" "$work/gates" <<'EOF'
import fs from "node:fs";
const [mappingPath, outputPath] = process.argv.slice(2);
const mapping = JSON.parse(fs.readFileSync(mappingPath, "utf8"));
const gates = [...new Set(Object.values(mapping.claims).map(claim => claim.gate))].sort();
if (gates.length !== 34) throw new Error(`expected 34 unique authoritative gates, got ${gates.length}`);
fs.writeFileSync(outputPath, `${gates.join("\n")}\n`);
EOF

while IFS= read -r gate; do
  printf 'UPB-v1 authoritative gate: %s\n' "$gate"
  "$repo/$gate"
done < "$work/gates"

"$repo/scripts/check-upb-v1-scorecard.sh"
echo 'UPB-v1 claimed cells: every mapped evidence gate passed'
