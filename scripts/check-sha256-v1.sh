#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-sha256.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

k0="$repo/bootstrap/seme-k0-linux-amd64"
module="$repo/modules/digest/sha256/v1"

(
    cd "$module"
    sha256sum -c digest.k0.sha256
    sha256sum -c digest.g1.sha256
    sha256sum -c digest.seme.sha256
)

(cd "$repo/reference/go" && go run ./cmd/sha256-program) > "$work/digest.s1"
cmp "$module/digest.s1" "$work/digest.s1"
"$k0" "$repo/bootstrap/s1-compiler.k0" "$work/digest.s1" "$work/digest.k0"
cmp "$module/digest.k0" "$work/digest.k0"
"$k0" "$repo/compiler/k0-module-lowerer.k0" "$module/digest.seme" \
    "$work/digest-lowered.k0"
cmp "$module/digest.k0" "$work/digest-lowered.k0"
"$k0" "$repo/compiler/kernel-wire-validator-located.k0" \
    "$module/digest.seme" "$work/digest-validated.seme"
cmp "$module/digest.seme" "$work/digest-validated.seme"

: > "$work/empty"
printf abc > "$work/abc"
head -c 55 /dev/zero > "$work/55"
head -c 56 /dev/zero > "$work/56"
head -c 64 /dev/zero > "$work/64"
head -c 1000 /dev/zero > "$work/1000"

for fixture in empty abc 55 56 64 1000; do
    "$k0" "$module/digest.k0" "$work/$fixture" "$work/$fixture.digest"
    actual=$(od -An -tx1 -v "$work/$fixture.digest" | tr -d ' \n')
    expected=$(sha256sum "$work/$fixture" | cut -d ' ' -f 1)
    test "$actual" = "$expected"
done

echo "Canonical SHA-256 v1 reproduction and conformance passed"
