#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-uab-02.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME
GOCACHE="$work/go-build"; export GOCACHE

# Direct source-bridge evidence exists for every UAB-02 family. This does not
# by itself establish frozen UAB-02 parity: the native observations below are
# not yet the exact boundary/adversarial vectors used by every target gate.
(cd "$repo/reference/go" && go test -count=1 -buildvcs=false ./goprojector \
  -run 'TestProjectsExistingCompositeCollectionsAndRecords|TestProjectsBytesOptionAndResultConstructors|TestRejectsMalformedCompositeSemantics|TestRejectsUnsupportedCanonicalExpression')

# Frozen standalone/Pulp evidence exists for every family, but several gates
# use different canonical programs or vectors from the direct Go evidence.
"$repo/scripts/check-uab-v1-go-01.sh"       # i64, bool, text
"$repo/scripts/check-execution-v17.sh"      # records
"$repo/scripts/check-execution-v21.sh"      # fixed arrays
"$repo/scripts/check-execution-v23.sh"      # slices
"$repo/scripts/check-execution-v30.sh"      # runtime-keyed maps
"$repo/scripts/check-execution-v31.sh"      # neutral Option schema rejection
"$repo/scripts/check-execution-v32.sh"      # bytes/match schema rejection
"$repo/scripts/check-composite-runtime-v32.sh" # bytes, Result, Option

echo "Go UAB-02 partial evidence passed; NOT CERTIFIED: native, canonical, and target observations are not yet directly linked for every required type family"
