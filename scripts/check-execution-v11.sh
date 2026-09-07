#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v11.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

k0="$repo/bootstrap/seme-k0-linux-amd64"
g1="$repo/compiler/g1-compiler.k0"
kernel="$repo/compiler/kernel-wire-validator.k0"
foundation="$repo/modules/foundation/v1/validator.k0"
execution="$repo/modules/execution/v11"

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
    cd "$repo/reference/go"
    go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget
    go run -buildvcs=false ./cmd/execution-module-v11 > "$work/module.g1"
    go build -buildvcs=false -o "$work/go-provider" ./cmd/go-provider
    go build -buildvcs=false -o "$work/go-lift" ./cmd/go-execution-lift
    go build -buildvcs=false -o "$work/structured-check" ./cmd/structured-v11-check
)

cmp "$execution/module.g1" "$work/module.g1"
(
    cd "$execution"
    sha256sum -c module.g1.sha256
    sha256sum -c module.seme.sha256
)
"$k0" "$g1" "$execution/module.g1" "$work/module.seme"
cmp "$execution/module.seme" "$work/module.seme"
"$k0" "$kernel" "$execution/module.seme"
"$k0" "$foundation" "$execution/module.seme"

cp -R "$repo/fixtures/go-execution-v11" "$work/project"
(cd "$work/project" && go test -buildvcs=false ./...)
"$work/go-provider" ingest \
    --project "$work/project" --module "$repo/modules/provider/v1/module.g1" \
    --out "$work/import"
"$work/go-lift" \
    --project "$work/project" --manifest "$work/import/manifest.json" \
    --module "$execution/module.g1" --function WithinAndEnabled \
    --profile structured-v8 --out "$work/program-a.g1"
"$work/go-lift" \
    --project "$work/project" --manifest "$work/import/manifest.json" \
    --module "$execution/module.g1" --function WithinAndEnabled \
    --profile structured-v8 --out "$work/program-b.g1"
cmp "$work/program-a.g1" "$work/program-b.g1"
"$k0" "$g1" "$work/program-a.g1" "$work/program.seme"
"$k0" "$kernel" "$work/program.seme"
"$k0" "$foundation" "$work/program.seme"
"$work/structured-check" "$work/program.seme"

# A BooleanLiteral field with unsigned shape remains structurally valid Kernel
# data but fails the canonical Foundation schema contract.
sed 's/fi 00000000000000000000000000009b00 tr/fi 00000000000000000000000000009b00 uu 1/' \
    "$work/program-a.g1" > "$work/non-boolean.g1"
"$k0" "$g1" "$work/non-boolean.g1" "$work/non-boolean.seme"
"$k0" "$kernel" "$work/non-boolean.seme"
expect_status 65 "$k0" "$foundation" "$work/non-boolean.seme"

version=2
while [ "$version" -le 10 ]; do
    (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/module-v$version.g1"
    cmp "$repo/modules/execution/v$version/module.g1" "$work/module-v$version.g1"
    version=$((version + 1))
done

echo "Core Execution v11: canonical short-circuit BooleanAnd passed"
