#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

k0="$repo/bootstrap/seme-k0-linux-amd64"
s1="$repo/bootstrap/s1-compiler.k0"
g1="$repo/compiler/g1-compiler.k0"
kernel="$repo/compiler/kernel-wire-validator.k0"
foundation="$repo/modules/foundation/v1/validator.k0"
lowerer="$repo/compiler/k0-module-lowerer.k0"
execution="$repo/modules/execution/v1"

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

go_provider="$work/go-provider"
go_lift="$work/go-execution-lift"
vectors="$work/execution-vector"
(
    cd "$repo/reference/go"
    go build -o "$go_provider" ./cmd/go-provider
    go build -o "$go_lift" ./cmd/go-execution-lift
    go build -o "$vectors" ./cmd/execution-vector
    go run ./cmd/execution-module > "$work/module.g1"
)

# Both the semantic vocabulary and its Seme-owned interpreter reproduce from
# retained construction sources and canonical executable graphs.
(
    cd "$execution"
    sha256sum -c module.g1.sha256
    sha256sum -c module.seme.sha256
    sha256sum -c interpreter.k0.sha256
    sha256sum -c interpreter.g1.sha256
    sha256sum -c interpreter.seme.sha256
)
cmp "$execution/module.g1" "$work/module.g1"
"$k0" "$g1" "$work/module.g1" "$work/module.seme"
cmp "$execution/module.seme" "$work/module.seme"
"$k0" "$kernel" "$execution/module.seme"
"$k0" "$foundation" "$execution/module.seme"

"$k0" "$s1" "$execution/interpreter.s1" "$work/interpreter.k0"
cmp "$execution/interpreter.k0" "$work/interpreter.k0"
"$k0" "$lowerer" "$execution/interpreter.seme" "$work/interpreter-lowered.k0"
cmp "$execution/interpreter.k0" "$work/interpreter-lowered.k0"
"$k0" "$kernel" "$execution/interpreter.seme"

# Lift one ordinary, parsed and type-checked Go function into language-neutral
# Core Execution entities. Repetition proves deterministic construction.
cp -R "$repo/fixtures/go-provider-v1" "$work/project"
(cd "$work/project" && go test ./...)
"$go_provider" ingest \
    --project "$work/project" --module "$repo/modules/provider/v1/module.g1" \
    --out "$work/import"
"$go_lift" \
    --project "$work/project" --manifest "$work/import/manifest.json" \
    --module "$execution/module.g1" --function Add --out "$work/program-a.g1"
"$go_lift" \
    --project "$work/project" --manifest "$work/import/manifest.json" \
    --module "$execution/module.g1" --function Add --out "$work/program-b.g1"
cmp "$work/program-a.g1" "$work/program-b.g1"
"$k0" "$g1" "$work/program-a.g1" "$work/program.seme"
"$k0" "$kernel" "$work/program.seme"
"$k0" "$foundation" "$work/program.seme"

# Runtime phase: the command inside this loop is only the frozen K0 executor,
# the Seme-authored interpreter, canonical program, and binary arguments. The
# Go oracle runs afterward and is not available to the execution itself.
index=0
for pair in \
    "20 22" \
    "-7 3" \
    "9223372036854775807 1" \
    "-9223372036854775808 -1" \
    "0 0"
do
    set -- $pair
    args="$work/args-$index.bin"
    result="$work/result-$index.bin"
    "$vectors" write "$1" "$2" "$args"
    "$k0" "$execution/interpreter.k0" "$work/program.seme" "$args" "$result"
    "$vectors" check "$1" "$2" "$result"
    index=$((index + 1))
done

# The lift and runtime fail closed outside the frozen profile.
cp -R "$work/project" "$work/unsupported"
sed 's/return a + b/return a - b/' "$work/project/add.go" > "$work/unsupported/add.go"
expect_status 65 "$go_lift" \
    --project "$work/unsupported" --manifest "$work/import/manifest.json" \
    --module "$execution/module.g1" --function Add --out "$work/unsupported.g1"
printf '\000' > "$work/bad-args.bin"
expect_status 65 "$k0" "$execution/interpreter.k0" \
    "$work/program.seme" "$work/bad-args.bin" "$work/bad-result.bin"

echo "Core Execution v1: exact Go lift and independent Seme execution passed"
