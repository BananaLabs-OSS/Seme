#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-kernel-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

a0="$repo/bootstrap/seme-a0-linux-amd64"
k0="$repo/bootstrap/seme-k0-linux-amd64"
frozen="$repo/compiler/kernel-wire-validator.k0"
validator="$repo/compiler/kernel-wire-validator-located.k0"

case "${1:-}" in
    '') reproduce=0 ;;
    --reproduce) reproduce=1 ;;
    *) echo "usage: $0 [--reproduce]" >&2; exit 64 ;;
esac

(
    cd "$repo/compiler"
    sha256sum -c kernel-wire-validator-located.g1.sha256
    sha256sum -c kernel-wire-validator-located.seme.sha256
    sha256sum -c kernel-wire-validator-located.k0.sha256
    sha256sum -c kernel-meta-bootstrap.seme.sha256
    sha256sum -c kernel-migrate-v0.seme.sha256
    sha256sum -c kernel-migrate-v0.k0.sha256
)

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

"$a0" "$repo/conformance/kernel-v1/obsolete-v0.a0" "$work/obsolete-v0.seme"
"$k0" "$repo/compiler/kernel-migrate-v0.k0" \
    "$work/obsolete-v0.seme" "$work/migrated-v1.seme"
"$k0" "$validator" "$work/migrated-v1.seme" "$work/migrated-valid.seme"
cmp "$work/migrated-v1.seme" "$work/migrated-valid.seme"

if [ "$reproduce" -eq 1 ]; then
    "$k0" "$repo/compiler/g1-compiler.k0" \
        "$repo/compiler/kernel-wire-validator-located.g1" "$work/validator.seme"
    cmp "$repo/compiler/kernel-wire-validator-located.seme" "$work/validator.seme"
    "$k0" "$repo/compiler/kernel-wire-validator.k0" "$work/validator.seme"
    "$k0" "$repo/compiler/k0-module-lowerer.k0" \
        "$work/validator.seme" "$work/validator.k0"
    cmp "$repo/compiler/kernel-wire-validator-located.k0" "$work/validator.k0"
fi

echo "Kernel v1: frozen hashes, 5 preservation fixtures, 7 diagnostic fixtures, and v0 migration passed"
