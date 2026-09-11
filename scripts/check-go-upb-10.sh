#!/bin/sh
# Cumulative Go UPB-10 seven-class authority and deployment gate.
# This is the sole script permitted to claim go/UPB-10.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb10.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"
export GOCACHE XDG_CACHE_HOME

# Preserve all seven evidence classes and the exact semantic round trip of the
# complete claimed predecessor before adding physical target authority.
"$repo/scripts/check-go-upb-09.sh"

# Reproduce and independently validate the two neutral contracts used by the
# additive placement layer.
"$repo/scripts/check-target-v1.sh"
"$repo/scripts/check-project-contract-v13.sh"

# Exercise language-neutral resolution, Go-profile closure discovery,
# authenticated catalog/launch records, source-free reopening, reports, and
# atomic publication units independently of the real fixture gate below.
(cd "$repo/reference/go" && go test -p=1 -count=1 \
  ./canonicalclosure ./targetplaninstance ./goprojectplacementadapter \
  ./projectv13instance ./goupb10pipeline ./goupb10deployment \
  ./goupb10bundle ./goupb10cmdload ./goupb10report \
  ./cmd/canonical-closure-check ./cmd/go-upb10-place \
  ./cmd/go-upb10-report ./cmd/go-upb10-policy-check)

# Two real independent deployments, exact-only rejection, closed canonical
# artifacts, and catalog/plan/project/VM/Pulp/native-island binding.
"$repo/scripts/check-go-upb-10-placement.sh"

printf 'Go UPB-10 seven-class target placement and deployment authority gate passes\n'
