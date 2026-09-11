#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-upb01.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME

"$repo/scripts/check-javascript-uab-01.sh"
(cd "$repo/reference/go" && go test -count=1 -buildvcs=false \
  ./projectsource ./projectbundle ./projectroundtrip ./projectemitter \
  ./sourceinventory ./cmd/project-source-discover ./cmd/project-assemble \
  ./cmd/source-inventory-emit ./cmd/project-roundtrip-publish)
"$repo/scripts/check-javascript-upb-01-source.sh"

echo 'JavaScript UPB-01: all seven evidence classes pass for the bounded ES-module project'
