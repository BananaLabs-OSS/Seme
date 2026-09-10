#!/bin/sh
# Cumulative Go UPB-09 pre-claim gate. Passing this scaffold is deliberately
# insufficient to map UPB-09: runtime parity, the 4,096-observation controlled
# corpus, projection/re-lift, and complete fixed-point evidence remain required.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb09.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"
export GOCACHE XDG_CACHE_HOME

# Preserve the complete claimed predecessor and reproduce the two additive
# neutral contracts independently.
"$repo/scripts/check-go-upb-08.sh"
"$repo/scripts/check-go-upb-09-fixture.sh"
"$repo/scripts/check-controlled-effects-v1.sh"
"$repo/scripts/check-project-contract-v12.sh"

# Exercise only presently implemented producer/consumer, authenticated
# instance, closed-world bundle, report, and command-boundary evidence. The
# heavyweight real-project integration test is intentionally not treated as
# proven by this scaffold.
(cd "$repo/reference/go" && go test -p=1 -count=1 \
  ./gocontrolledeffectsmanifest ./gocontrolledeffectsadapter \
  ./projectv12instance ./goupb09pipeline ./goupb09bundle ./goupb09report \
  ./goupb09cmdload ./cmd/go-upb09-build ./cmd/go-upb09-project \
  ./cmd/go-upb09-report &&
  go test -p=1 -count=1 ./controlledeffectsinstance \
    -run '^(TestReplayDigestIsDeterministicAndMeaningSensitive|TestModelRejectsOwnershipSignaturePurityAndReplayAdversaries|TestTranscriptBoundExactAndPlusOne|TestZeroAuthorityAndArtifactAreRejected|TestBooleanEncodingIsExact)$')

printf 'Go UPB-09 implemented pre-claim evidence passes; UPB-09 remains unclaimed\n'
