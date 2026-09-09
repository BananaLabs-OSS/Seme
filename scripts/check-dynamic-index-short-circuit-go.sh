#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-dynamic-index-go.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"
XDG_CACHE_HOME="$work/xdg-cache"
export GOCACHE XDG_CACHE_HOME
(cd "$repo/reference/go" && go test -buildvcs=false ./canonicaleval ./wasmtarget -run 'DynamicIndex')
echo 'Dynamic index Go evidence: evaluator bounds/short-circuit and target structure/descriptors reject safely'
