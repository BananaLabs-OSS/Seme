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
    sha256sum -c reporter.k0.sha256
    sha256sum -c reporter.g1.sha256
    sha256sum -c reporter.seme.sha256
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
"$k0" "$repo/bootstrap/s1-compiler.k0" "$foundation/reporter.s1" \
    "$work/foundation-reporter.k0"
cmp "$foundation/reporter.k0" "$work/foundation-reporter.k0"
"$k0" "$repo/compiler/k0-module-lowerer.k0" "$foundation/reporter.seme" \
    "$work/foundation-reporter-lowered.k0"
cmp "$foundation/reporter.k0" "$work/foundation-reporter-lowered.k0"
"$k0" "$kernel" "$foundation/reporter.seme" \
    "$work/foundation-reporter.validated.seme"
cmp "$foundation/reporter.seme" "$work/foundation-reporter.validated.seme"
"$k0" "$foundation/validator.k0" "$foundation/module.seme" \
    "$work/foundation-semantic.validated.seme"
cmp "$foundation/module.seme" "$work/foundation-semantic.validated.seme"
"$k0" "$foundation/reporter.k0" "$foundation/module.seme" \
    "$work/foundation-report.seme"
"$k0" "$kernel" "$work/foundation-report.seme"
(cd "$repo/reference/go" && go run ./cmd/foundation-report-check \
    "$work/foundation-report.seme" 61 61 0 0)

# A module whose declarations are unavailable remains preserved, while a known
# future schema version is preserved rather than falsely certified or rejected.
"$k0" "$foundation/validator.k0" "$patch/module.seme" \
    "$work/patch-unavailable.seme"
cmp "$patch/module.seme" "$work/patch-unavailable.seme"
"$k0" "$foundation/reporter.k0" "$patch/module.seme" \
    "$work/patch-unavailable-report.seme"
(cd "$repo/reference/go" && go run ./cmd/foundation-report-check \
    "$work/patch-unavailable-report.seme" 10 0 0 10)
awk '$1 == "en" && $2 == "00000000000000000000000000000011" { $4 = 2 } { print }' \
    "$foundation/module.g1" > "$work/foundation-future.g1"
"$k0" "$g1" "$work/foundation-future.g1" "$work/foundation-future.seme"
"$k0" "$foundation/validator.k0" "$work/foundation-future.seme" \
    "$work/foundation-future.preserved.seme"
cmp "$work/foundation-future.seme" "$work/foundation-future.preserved.seme"
"$k0" "$foundation/reporter.k0" "$work/foundation-future.seme" \
    "$work/foundation-future-report.seme"
(cd "$repo/reference/go" && go run ./cmd/foundation-report-check \
    "$work/foundation-future-report.seme" 61 60 1 0)

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
expect_status 65 "$k0" "$foundation/validator.k0" \
    "$work/foundation-invalid-since.seme" \
    "$work/foundation-invalid-since.report.seme"
"$k0" "$kernel" "$work/foundation-invalid-since.report.seme" \
    "$work/foundation-invalid-since.report.validated.seme"
cmp "$work/foundation-invalid-since.report.seme" \
    "$work/foundation-invalid-since.report.validated.seme"

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
expect_status 65 "$k0" "$foundation/validator.k0" \
    "$work/foundation-invalid-shape.seme" \
    "$work/foundation-invalid-shape.report.seme"
"$k0" "$kernel" "$work/foundation-invalid-shape.report.seme" \
    "$work/foundation-invalid-shape.report.validated.seme"
cmp "$work/foundation-invalid-shape.report.seme" \
    "$work/foundation-invalid-shape.report.validated.seme"

(
    cd "$patch"
    sha256sum -c module.g1.sha256
    sha256sum -c module.seme.sha256
    sha256sum -c apply-candidate.k0.sha256
    sha256sum -c apply-candidate.g1.sha256
    sha256sum -c apply-candidate.seme.sha256
)

(
    cd "$repo/reference/go"
    go run ./cmd/patch-module
) > "$work/patch-v1.g1"
cmp "$patch/module.g1" "$work/patch-v1.g1"
"$k0" "$g1" "$patch/module.g1" "$work/patch-v1.seme"
cmp "$patch/module.seme" "$work/patch-v1.seme"
"$k0" "$kernel" "$work/patch-v1.seme" "$work/patch-v1.validated.seme"
cmp "$work/patch-v1.seme" "$work/patch-v1.validated.seme"
"$k0" "$repo/bootstrap/s1-compiler.k0" "$patch/apply-candidate.s1" \
    "$work/patch-apply-candidate.k0"
cmp "$patch/apply-candidate.k0" "$work/patch-apply-candidate.k0"
"$k0" "$repo/compiler/k0-module-lowerer.k0" \
    "$patch/apply-candidate.seme" "$work/patch-apply-candidate-lowered.k0"
cmp "$patch/apply-candidate.k0" "$work/patch-apply-candidate-lowered.k0"
"$k0" "$kernel" "$patch/apply-candidate.seme" \
    "$work/patch-apply-candidate.validated.seme"
cmp "$patch/apply-candidate.seme" \
    "$work/patch-apply-candidate.validated.seme"

for mode in valid stale precondition duplicate; do
    (
        cd "$repo/reference/go"
        go run ./cmd/patch-fixture "$mode"
    ) > "$work/patch-$mode.g1"
    "$k0" "$g1" "$work/patch-$mode.g1" "$work/patch-$mode.seme"
    "$k0" "$repo/compiler/kernel-wire-validator.k0" \
        "$work/patch-$mode.seme"
    "$k0" "$foundation/validator.k0" "$work/patch-$mode.seme"
done

"$k0" "$patch/apply-candidate.k0" "$work/patch-valid.seme" \
    "$work/patch-valid.candidate.seme"
"$k0" "$repo/compiler/kernel-wire-validator.k0" \
    "$work/patch-valid.candidate.seme"
"$k0" "$foundation/validator.k0" "$work/patch-valid.candidate.seme"

for mode in stale precondition duplicate; do
    cp "$work/patch-$mode.seme" "$work/patch-$mode.unchanged.seme"
    cp "$work/patch-$mode.unchanged.seme" "$work/patch-$mode.output.seme"
    expect_status 65 "$k0" "$patch/apply-candidate.k0" \
        "$work/patch-$mode.seme" "$work/patch-$mode.output.seme"
    cmp "$work/patch-$mode.unchanged.seme" \
        "$work/patch-$mode.output.seme"
done

(cd "$repo/reference/go" && go test ./...)

(cd "$repo/reference/go" && go run ./cmd/k0-lift \
    "$repo/compiler/kernel-wire-validator.k0") > "$work/kernel-validator-lifted.g1"
"$k0" "$g1" "$work/kernel-validator-lifted.g1" "$work/kernel-validator-lifted.seme"
cmp "$repo/compiler/kernel-wire-validator.seme" "$work/kernel-validator-lifted.seme"

echo "Semantic modules: Foundation v1 and Patch v1 canonical graphs and reference conformance passed"
