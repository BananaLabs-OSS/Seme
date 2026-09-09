#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
node "$repo/scripts/check-upb-v1-scorecard.mjs" "$repo/conformance/upb-v1/scorecard.json"
echo 'UPB-v1 scorecard shape only; no cell is claimed and no evidence gate has been executed'
