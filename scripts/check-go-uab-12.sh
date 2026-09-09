#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-uab12.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go"; export GOCACHE
cd "$repo"

# The JavaScript application is the declared shared canonical seed, not a Go
# behavior oracle. Go must open this exact graph as native source and preserve
# its canonical bytes on re-import.
node reference/js/javascript-package-provider-cli.mjs \
  --files fixtures/javascript-uab-11/application.js,fixtures/javascript-uab-11/policy.js \
  --module modules/execution/v35/module.g1 --package seme.uab11/application \
  --entry Apply --revision 1 --out "$work/seed.g1"
bootstrap/seme-k0-linux-amd64 compiler/g1-compiler.k0 "$work/seed.g1" "$work/seed.seme"

(cd reference/go &&
  go test -count=1 -buildvcs=false ./goprojector ./goprovider ./cmd/go-session-proof &&
  go build -buildvcs=false -o "$work/projector" ./cmd/go-projector &&
  go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof)
mkdir -p "$work/project"
"$work/projector" -package application "$work/seed.g1" "$work/project/application.go"
cp fixtures/go-uab-11/application/application_test.go "$work/project/application_test.go"
cp fixtures/go-uab-11/go.mod "$work/project/go.mod"
(cd "$work/project" && go test -count=1 -buildvcs=false ./...)

"$work/session" --module modules/execution/v35/module.g1 --project "$work/project" \
  --package seme.uab11/application --entry Apply --revision 1 --out "$work/relift.g1"
cmp "$work/seed.g1" "$work/relift.g1"

# A semantic edit cannot retain the old envelope. Removing provenance permits
# an honest fresh lift, while keeping stale provenance rejects.
cp -R "$work/project" "$work/edited"
sed -i '0,/Error: 1/s//Error: 9/' "$work/edited/application.go"
if "$work/session" --module modules/execution/v35/module.g1 --project "$work/edited" \
  --package seme.uab11/application --entry Apply --revision 1 --out "$work/edited.g1" \
  >"$work/edited.out" 2>"$work/edited.log"; then
  echo 'Go UAB-12 accepted source edited beneath projection envelope' >&2; exit 1
fi
rg -q 'go_projection.envelope_source_mismatch' "$work/edited.log"

echo 'Go UAB-12: shared canonical seed projects to compiling native Go, passes 128x16 commands, re-lifts byte-identically, and rejects forged provenance'
