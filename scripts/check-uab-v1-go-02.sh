#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-uab-02.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME
GOCACHE="$work/go-build"; export GOCACHE

# Direct source-bridge and shared scalar/composite vector evidence.
(cd "$repo/reference/go" && go test -count=1 -buildvcs=false ./goprojector \
  -run 'TestProjectsExistingCompositeCollectionsAndRecords|TestProjectsBytesOptionAndResultConstructors|TestRejectsMalformedCompositeSemantics|TestRejectsUnsupportedCanonicalExpression')
"$repo/scripts/check-go-uab-02-scalars.sh"
"$repo/scripts/check-go-uab-02-composite.sh"
"$repo/scripts/check-go-uab-02-aggregate.sh"

# Frozen collection standalone/Pulp evidence uses the same source fixtures and
# observation vectors now exercised by the direct projector tests above.
"$repo/scripts/check-uab-v1-go-01.sh"       # i64, bool, text
"$repo/scripts/check-execution-v17.sh"      # records
"$repo/scripts/check-execution-v21.sh"      # fixed arrays
"$repo/scripts/check-execution-v23.sh"      # slices
"$repo/scripts/check-execution-v30.sh"      # runtime-keyed maps
"$repo/scripts/check-execution-v31.sh"      # neutral Option schema rejection
"$repo/scripts/check-execution-v32.sh"      # bytes/match schema rejection
"$repo/scripts/check-composite-runtime-v32.sh" # bytes, Result, Option

node -e '
  const report = require(process.argv[1]);
  const expected = ["lift", "native_parity", "target_parity", "projection_round_trip", "rejection"];
  if (JSON.stringify(report.languages.go["UAB-02"]) !== JSON.stringify(expected)) process.exit(1);
' "$repo/conformance/uab-v1/scorecard.json"

echo "UAB-v1 Go UAB-02: exact shared observations passed all five evidence categories for every required type family"
