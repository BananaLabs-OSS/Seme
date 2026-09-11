#!/bin/sh
# Cumulative JavaScript UPB-11 seven-class semantic project reconciliation gate.
# This is the sole script permitted to claim javascript/UPB-11.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-upb11.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"
export GOCACHE XDG_CACHE_HOME

"$repo/scripts/check-javascript-upb-10.sh"
"$repo/scripts/check-language-service-v1.sh"
"$repo/scripts/check-project-contract-v14.sh"
(cd "$repo/reference/js" && npm test)
(cd "$repo/reference/go" && go test -p=1 -count=1 \
  ./goupb09bundle ./patchinstance ./projectv14instance \
  ./cmd/project-v12-bundle ./cmd/project-v14-finalize)
"$repo/scripts/check-javascript-upb-11-reconciliation.sh"
"$repo/scripts/check-javascript-upb-11-project.sh"
printf 'JavaScript UPB-11 seven-class transactional project reconciliation authority gate passes\n'
