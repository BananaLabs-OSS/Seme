#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-kernel-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

a0="$repo/bootstrap/seme-a0-linux-amd64"
k0="$repo/bootstrap/seme-k0-linux-amd64"
frozen="$repo/compiler/kernel-wire-validator.k0"
validator="$repo/compiler/kernel-wire-validator-located.k0"

valid='minimal values forward-reference external-counter kernel-meta-minimal'
invalid='kernel-meta-incomplete dangling-reference duplicate-entities trailing-byte unknown-value-tag unsorted-parents noncanonical-version'

for name in $valid; do
    "$a0" "$repo/conformance/kernel-v1/$name.a0" "$work/$name.seme"
    "$k0" "$validator" "$work/$name.seme" "$work/$name.out"
    cmp "$work/$name.seme" "$work/$name.out"
done

for name in $invalid; do
    "$a0" "$repo/conformance/kernel-v1/$name.a0" "$work/$name.seme"
    status=0
    "$k0" "$validator" "$work/$name.seme" "$work/$name.report" || status=$?
    if [ "$status" -ne 65 ]; then
        echo "$name: validator status $status, want 65" >&2
        exit 1
    fi
    "$k0" "$frozen" "$work/$name.report"
done

echo "Kernel v1 baseline: 5 valid preservation and 7 invalid diagnostic fixtures passed"
