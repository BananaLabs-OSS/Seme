#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

(
    cd "$repo/reference/go"
    go test -buildvcs=false -run '^TestPureCertificate' ./wasmtarget
)

echo "Core v12 certification: membership, typing, purity, and bounded graph rejection passed"
