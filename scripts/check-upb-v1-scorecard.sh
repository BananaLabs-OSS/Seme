#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
node "$repo/scripts/check-upb-v1-scorecard.mjs" "$repo/conformance/upb-v1/scorecard.json" "$repo/conformance/upb-v1/evidence-gates.json"
echo 'UPB-v1 scorecard claims have exact gate mappings; this shape check does not execute those gates'
