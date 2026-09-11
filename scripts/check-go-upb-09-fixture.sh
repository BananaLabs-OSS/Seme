#!/bin/sh
# Native fixture partition for Go UPB-09. This does not establish
# canonical, Wasm, Pulp, replay-corpus, projection, or fixed-point parity.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb09-fixture.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; export GOCACHE
proxy="$repo/fixtures/go-upb03-offline-proxy"

"$repo/scripts/materialize-go-upb09-fixture.sh" "$work/project-a"
"$repo/scripts/materialize-go-upb09-fixture.sh" "$work/project-b"
diff -ru "$work/project-a" "$work/project-b"
(cd "$work/project-a" &&
  GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off \
  go test -race -count=1 -buildvcs=false ./...)

printf 'Go UPB-09 fixture partition: deterministic materialization and native race tests pass\n'
