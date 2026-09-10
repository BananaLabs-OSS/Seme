#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
destination=${1:?destination required}
"$repo/scripts/materialize-go-upb05-fixture.sh" "$destination"
mkdir -m 700 "$destination/resource" "$destination/resources"
cp "$repo/fixtures/go-upb06-resource-overlay/resource/resource.go" "$destination/resource/resource.go"
cp "$repo/fixtures/go-upb06-resource-overlay/resource/resource_test.go" "$destination/resource/resource_test.go"
cp "$repo/fixtures/go-upb06-resource-overlay/service/resource.go" "$destination/service/resource.go"
cp "$repo/fixtures/go-upb06-resource-overlay/service/resource_test.go" "$destination/service/resource_test.go"
cp "$repo/fixtures/go-upb06-resource-overlay/service/native_resource_corpus_test.go" "$destination/service/native_resource_corpus_test.go"
cp "$repo/fixtures/go-upb06-resource-overlay/resources.json" "$destination/resources.json"
cp "$repo/fixtures/go-upb06-resource-overlay/resources/notice.txt" "$destination/resources/notice.txt"
base64 -d < "$repo/fixtures/go-upb06-resource-overlay/resources/marker.bin.base64" > "$destination/resources/marker.bin"
