#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-ordered-transport-runtime.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
(cd "$repo/reference/go" && GOCACHE="$work/cache" go test -count=1 ./orderedtransportruntime ./orderedtransportinstance ./contractcatalog)
echo 'Ordered Transport host boundary: authenticated profile, strict fragmented/coalesced framing, exact bounds, authorized receive/send, independent accepted-byte evidence, and committed retry pass'
