#!/bin/sh
# Unclaimed cumulative UPB-08 development gate. This script intentionally
# requires the scorecard cell and evidence-gate mapping to remain empty.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb08.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME

# Cumulative predecessor and independently attributable UPB08 partitions.
"$repo/scripts/check-go-upb-07.sh"
"$repo/scripts/check-go-upb-08-fixture.sh"
"$repo/scripts/check-ordered-transport-v1.sh"
"$repo/scripts/check-project-contract-v11.sh"
"$repo/scripts/check-go-upb-08-runtime.sh"
"$repo/scripts/check-go-upb-08-port-runtime.sh"
"$repo/scripts/check-go-upb-08-pulp-placement.sh"

# Producer/consumer, closed-world report, projection/re-lift fixed point, and
# atomic rejection are exercised by focused packages. goupb08bundle includes
# the real cumulative source-free load rather than a fabricated unit graph.
(cd "$repo/reference/go" && go test -count=1 \
  ./goorderedtransportmanifest ./goorderedtransportadapter \
  ./orderedtransportinstance ./projectv11instance \
  ./goupb08pipeline ./goupb08bundle ./goupb08report ./goupb08portruntime \
  ./cmd/go-upb08-build ./cmd/go-upb08-project ./cmd/go-upb08-report)

# A development gate cannot silently convert itself into a mapped claim.
node - "$repo/conformance/upb-v1/scorecard.json" "$repo/conformance/upb-v1/evidence-gates.json" <<'NODE'
const fs=require("fs"),score=JSON.parse(fs.readFileSync(process.argv[2])),gates=JSON.parse(fs.readFileSync(process.argv[3]));
if(JSON.stringify(score.languages.go["UPB-08"])!=="[]")throw Error("unreviewed UPB-08 scorecard claim");
if(Object.prototype.hasOwnProperty.call(gates.claims,"go/UPB-08"))throw Error("unreviewed UPB-08 gate mapping");
NODE
printf 'Go UPB-08 unclaimed cumulative development gate passes; scorecard and evidence mapping remain unchanged\n'
