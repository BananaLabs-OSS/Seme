#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
destination=${1:?destination required}
"$repo/scripts/materialize-go-upb08-fixture.sh" "$destination"
mkdir -m 700 "$destination/controlled"
cp "$repo/fixtures/go-upb09-effects-overlay/controlled/source.go" "$destination/controlled/source.go"
cp "$repo/fixtures/go-upb09-effects-overlay/controlled/source_test.go" "$destination/controlled/source_test.go"
cp "$repo/fixtures/go-upb09-effects-overlay/streamservice/controlled.go" "$destination/streamservice/controlled.go"
cp "$repo/fixtures/go-upb09-effects-overlay/streamservice/controlled_test.go" "$destination/streamservice/controlled_test.go"
cp "$repo/fixtures/go-upb09-effects-overlay/streamservice/native_effects_corpus_test.go" "$destination/streamservice/native_effects_corpus_test.go"
cp "$repo/fixtures/go-upb09-effects-overlay/controlled-effects-selection.json" "$destination/controlled-effects-selection.json"
