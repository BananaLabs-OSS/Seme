#!/bin/sh
# Cumulative JavaScript UPB-10 seven-class authority and deployment gate.
# This is the sole script permitted to claim javascript/UPB-10.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-upb10.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"
export GOCACHE XDG_CACHE_HOME

"$repo/scripts/check-javascript-upb-09.sh"
"$repo/scripts/check-target-v1.sh"
"$repo/scripts/check-project-contract-v13.sh"
(cd "$repo/reference/go" && go test -p=1 -count=1 \
  ./canonicalclosure ./targetplaninstance ./goprojectplacementadapter \
  ./projectv13instance ./goupb10deployment ./goupb10bundle \
  ./goupb10cmdload ./goupb10report ./goupb09bundle \
  ./cmd/canonical-closure-check ./cmd/project-v12-compose \
  ./cmd/go-upb10-place ./cmd/go-upb10-report ./cmd/go-upb10-policy-check)
"$repo/scripts/check-javascript-upb-10-placement.sh"
printf 'JavaScript UPB-10 seven-class target placement and deployment authority gate passes\n'
