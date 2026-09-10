#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb05-fixture.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; export GOCACHE
"$repo/scripts/materialize-go-upb05-fixture.sh" "$work/project"
(cd "$work/project" && GOTOOLCHAIN=local GOPROXY="file://$repo/fixtures/go-upb03-offline-proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./...)
if rg -n 'os\.Getenv|os\.LookupEnv|func init\(|\btime\.|\brand\.|^var [A-Za-z_][A-Za-z0-9_]* =' "$work/project" --glob '*.go'; then
  echo 'Go UPB-05 fixture contains an ambient or global initialization mechanism' >&2
  exit 1
fi
echo 'Go UPB-05 fixture: cumulative 2,048 corpus and explicit configuration/lifecycle matrix pass'
