#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
destination=${1:?destination required}
"$repo/scripts/materialize-go-upb04-fixture.sh" "$destination"
mkdir -m 700 "$destination/configuration" "$destination/service"
cp "$repo/fixtures/go-upb05-configuration-overlay/configuration/configuration.go" "$destination/configuration/configuration.go"
cp "$repo/fixtures/go-upb05-configuration-overlay/policy/initialization.go" "$destination/policy/initialization.go"
cp "$repo/fixtures/go-upb05-configuration-overlay/service/service.go" "$destination/service/service.go"
cp "$repo/fixtures/go-upb05-configuration-overlay/service/service_test.go" "$destination/service/service_test.go"
cp "$repo/fixtures/go-upb05-configuration-overlay/service/native_corpus_test.go" "$destination/service/native_corpus_test.go"
