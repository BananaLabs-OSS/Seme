#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-uab11-source.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go"; export GOCACHE

(cd "$repo/fixtures/go-uab-11" && go test -count=1 -buildvcs=false ./...)
(cd "$repo/reference/go" &&
  go test -count=1 -buildvcs=false ./goprovider ./goprojector ./canonicaleval &&
  go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof &&
  go build -buildvcs=false -o "$work/projector" ./cmd/go-projector &&
  go build -buildvcs=false -o "$work/observe" ./cmd/canonical-observe)

"$work/session" --module "$repo/modules/execution/v35/module.g1" \
  --project "$repo/fixtures/go-uab-11" --package example.test/go-uab-11/application \
  --entry Apply --revision 1 --out "$work/program.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" \
  "$work/program.g1" "$work/program.seme"

node "$repo/reference/js/javascript-uab-11-canonical-corpus.mjs" \
  "$repo/fixtures/javascript-uab-11/application.js" "$work/requests.jsonl" "$work/expected.jsonl"
"$work/observe" "$work/program.seme" < "$work/requests.jsonl" > "$work/canonical.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/expected.jsonl" "$work/canonical.jsonl"
test "$(wc -l < "$work/canonical.jsonl" | tr -d ' ')" -eq 2048

"$work/projector" -package application "$work/program.g1" "$work/projected.go"
mkdir -p "$work/projected/application"
cp "$work/projected.go" "$work/projected/application/application.go"
cp "$repo/fixtures/go-uab-11/application/application_test.go" "$work/projected/application/application_test.go"
cp "$repo/fixtures/go-uab-11/go.mod" "$work/projected/go.mod"
(cd "$work/projected" && go test -count=1 -buildvcs=false ./...)
"$work/session" --module "$repo/modules/execution/v35/module.g1" \
  --project "$work/projected" --package example.test/go-uab-11/application \
  --entry Apply --revision 1 --out "$work/relift.g1"
cmp "$work/program.g1" "$work/relift.g1"

# Removing the required immutable map clone changes alias semantics and must
# not be guessed into a canonical update.
cp -R "$repo/fixtures/go-uab-11" "$work/nearby"
sed -i 's/output := maps.Clone(input)/output := input; _ = maps.Clone(input)/' "$work/nearby/application/application.go"
if "$work/session" --module "$repo/modules/execution/v35/module.g1" \
  --project "$work/nearby" --package example.test/go-uab-11/application \
  --entry Apply --revision 1 --out "$work/nearby.g1" >"$work/nearby.out" 2>"$work/nearby.log"; then
  echo 'Go UAB-11 accepted aliasing map update' >&2; exit 1
fi
rg -q 'expression.unsupported_function_literal_call' "$work/nearby.log"

echo 'Go UAB-11 source: native 128x16, unified lift, canonical 2048, projection, byte-identical re-lift, and source rejection pass'
