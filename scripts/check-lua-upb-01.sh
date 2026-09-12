#!/bin/sh
# Sole cumulative gate permitted to claim lua/UPB-01.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-lua-upb01.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
"$repo/scripts/check-lua-uab-11.sh"
(cd "$repo/reference/lua" && node --test lua-project-manifest.test.mjs)
(cd "$repo/reference/go" && go test -p=1 -count=1 ./projectsource ./projectbundle ./projectroundtrip ./projectemitter ./sourceinventory ./cmd/project-source-discover ./cmd/project-assemble ./cmd/source-inventory-emit ./cmd/project-roundtrip-publish)
"$repo/scripts/check-lua-upb-01-source.sh"
printf 'Lua UPB-01 all seven project evidence classes pass\n'
