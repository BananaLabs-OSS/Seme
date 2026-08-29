#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-modules.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

k0="$repo/bootstrap/seme-k0-linux-amd64"
g1="$repo/compiler/g1-compiler.k0"
kernel="$repo/compiler/kernel-wire-validator-located.k0"
patch="$repo/modules/patch/v1"
foundation="$repo/modules/foundation/v1"

expect_status() {
    expected=$1
    shift
    set +e
    "$@"
    actual=$?
    set -e
    if [ "$actual" -ne "$expected" ]; then
        echo "expected status $expected, got $actual: $*" >&2
        exit 1
    fi
}

(
    cd "$foundation"
    sha256sum -c module.g1.sha256
    sha256sum -c module.seme.sha256
)

(cd "$repo/reference/go" && go run ./cmd/foundation-module) > "$work/foundation-v1.g1"
cmp "$foundation/module.g1" "$work/foundation-v1.g1"
"$k0" "$g1" "$work/foundation-v1.g1" "$work/foundation-v1.seme"
cmp "$foundation/module.seme" "$work/foundation-v1.seme"
"$k0" "$kernel" "$work/foundation-v1.seme" "$work/foundation-v1.validated.seme"
cmp "$work/foundation-v1.seme" "$work/foundation-v1.validated.seme"

(
    cd "$foundation"
    sha256sum -c validator.k0.sha256
    sha256sum -c validator.g1.sha256
    sha256sum -c validator.seme.sha256
)
"$k0" "$repo/bootstrap/s1-compiler.k0" "$foundation/validator.s1" \
    "$work/foundation-validator.k0"
cmp "$foundation/validator.k0" "$work/foundation-validator.k0"
"$k0" "$repo/compiler/k0-module-lowerer.k0" "$foundation/validator.seme" \
    "$work/foundation-validator-lowered.k0"
cmp "$foundation/validator.k0" "$work/foundation-validator-lowered.k0"
"$k0" "$kernel" "$foundation/validator.seme" \
    "$work/foundation-validator.validated.seme"
cmp "$foundation/validator.seme" "$work/foundation-validator.validated.seme"
"$k0" "$foundation/validator.k0" "$foundation/module.seme" \
    "$work/foundation-semantic.validated.seme"
cmp "$foundation/module.seme" "$work/foundation-semantic.validated.seme"

# A module whose declarations are unavailable remains preserved, while a known
# future schema version is preserved rather than falsely certified or rejected.
"$k0" "$foundation/validator.k0" "$patch/module.seme" \
    "$work/patch-unavailable.seme"
cmp "$patch/module.seme" "$work/patch-unavailable.seme"
awk '$1 == "en" && $2 == "00000000000000000000000000000011" { $4 = 2 } { print }' \
    "$foundation/module.g1" > "$work/foundation-future.g1"
"$k0" "$g1" "$work/foundation-future.g1" "$work/foundation-future.seme"
"$k0" "$foundation/validator.k0" "$work/foundation-future.seme" \
    "$work/foundation-future.preserved.seme"
cmp "$work/foundation-future.seme" "$work/foundation-future.preserved.seme"

# Both declaration malformation and a declared value-shape violation are
# structurally valid Kernel graphs but must reject semantically.
awk '
    $1 == "en" { field = ($2 == "00000000000000000000000000000100") }
    field && $2 == "00000000000000000000000000000113" { $4 = 0 }
    { print }
' "$foundation/module.g1" > "$work/foundation-invalid-since.g1"
"$k0" "$g1" "$work/foundation-invalid-since.g1" \
    "$work/foundation-invalid-since.seme"
"$k0" "$repo/compiler/kernel-wire-validator.k0" \
    "$work/foundation-invalid-since.seme"
expect_status 65 "$k0" "$foundation/validator.k0" \
    "$work/foundation-invalid-since.seme"

awk '
    $1 == "en" { field = ($2 == "00000000000000000000000000000100") }
    field && $2 == "00000000000000000000000000002000" { $4 = 2 }
    { print }
' "$foundation/module.g1" > "$work/foundation-invalid-shape.g1"
"$k0" "$g1" "$work/foundation-invalid-shape.g1" \
    "$work/foundation-invalid-shape.seme"
"$k0" "$repo/compiler/kernel-wire-validator.k0" \
    "$work/foundation-invalid-shape.seme"
expect_status 65 "$k0" "$foundation/validator.k0" \
    "$work/foundation-invalid-shape.seme"

(
    cd "$patch"
    sha256sum -c module.g1.sha256
    sha256sum -c module.seme.sha256
)

"$k0" "$g1" "$patch/module.g1" "$work/patch-v1.seme"
cmp "$patch/module.seme" "$work/patch-v1.seme"
"$k0" "$kernel" "$work/patch-v1.seme" "$work/patch-v1.validated.seme"
cmp "$work/patch-v1.seme" "$work/patch-v1.validated.seme"

(cd "$repo/reference/go" && go test ./...)

(cd "$repo/reference/go" && go run ./cmd/k0-lift \
    "$repo/compiler/kernel-wire-validator.k0") > "$work/kernel-validator-lifted.g1"
"$k0" "$g1" "$work/kernel-validator-lifted.g1" "$work/kernel-validator-lifted.seme"
cmp "$repo/compiler/kernel-wire-validator.seme" "$work/kernel-validator-lifted.seme"

echo "Semantic modules: Foundation v1 and Patch v1 canonical graphs and reference conformance passed"
