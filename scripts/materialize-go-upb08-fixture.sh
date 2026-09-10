#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
destination=${1:?destination required}
"$repo/scripts/materialize-go-upb07-fixture.sh" "$destination"
mkdir -m 700 "$destination/transport" "$destination/streamservice"
cp "$repo/fixtures/go-upb08-transport-overlay/transport/stream.go" "$destination/transport/stream.go"
cp "$repo/fixtures/go-upb08-transport-overlay/transport/stream_test.go" "$destination/transport/stream_test.go"
cp "$repo/fixtures/go-upb08-transport-overlay/transport/native_corpus_test.go" "$destination/transport/native_corpus_test.go"
cp "$repo/fixtures/go-upb08-transport-overlay/streamservice/adapter.go" "$destination/streamservice/adapter.go"
cp "$repo/fixtures/go-upb08-transport-overlay/streamservice/adapter_test.go" "$destination/streamservice/adapter_test.go"
