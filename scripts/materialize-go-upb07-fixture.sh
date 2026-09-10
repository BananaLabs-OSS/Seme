#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
destination=${1:?destination required}
"$repo/scripts/materialize-go-upb06-fixture.sh" "$destination"
mkdir -m 700 "$destination/state" "$destination/persistence"
cp "$repo/fixtures/go-upb07-durable-overlay/state/state.go" "$destination/state/state.go"
cp "$repo/fixtures/go-upb07-durable-overlay/state/state_test.go" "$destination/state/state_test.go"
cp "$repo/fixtures/go-upb07-durable-overlay/persistence/planner.go" "$destination/persistence/planner.go"
cp "$repo/fixtures/go-upb07-durable-overlay/persistence/planner_test.go" "$destination/persistence/planner_test.go"
cp "$repo/fixtures/go-upb07-durable-overlay/persistence/native_corpus_test.go" "$destination/persistence/native_corpus_test.go"
