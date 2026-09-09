#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
node "$repo/reference/js/uab-v1-scorecard.mjs" "$repo/conformance/uab-v1/scorecard.json"
echo 'scorecard shape only; run scripts/check-uab-v1-complete-profile.sh to execute all claimed evidence'
