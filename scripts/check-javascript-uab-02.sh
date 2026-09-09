#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

(cd "$repo/reference/js" && npm test)
"$repo/scripts/check-execution-v17.sh" # Boolean/text/records
"$repo/scripts/check-execution-v19.sh" # signed i64 boundaries and rejection
"$repo/scripts/check-execution-v21.sh" # fixed arrays and shape rejection
"$repo/scripts/check-execution-v23.sh" # runtime slices
"$repo/scripts/check-execution-v30.sh" # runtime-keyed maps
"$repo/scripts/check-javascript-composite-v32.sh" # bytes, Result, Option, total matches

echo "JavaScript UAB-02 partial evidence: all value-family prerequisites pass, but shared cross-path vectors remain incomplete; cell not certified"
