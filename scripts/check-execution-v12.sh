#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v12.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

k0="$repo/bootstrap/seme-k0-linux-amd64"
g1="$repo/compiler/g1-compiler.k0"
kernel="$repo/compiler/kernel-wire-validator.k0"
foundation="$repo/modules/foundation/v1/validator.k0"
execution="$repo/modules/execution/v12"
target="$repo/targets/wasm/pulp-function-v1"

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

if [ ! -f "$pulp_repo/cmd/pulp-seme-function-proof/main.go" ]; then
    echo "Pulp generic function proof runner unavailable at $pulp_repo" >&2
    exit 65
fi

(
    cd "$repo/reference/go"
    go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget
    go run -buildvcs=false ./cmd/execution-module-v12 > "$work/module.g1"
    go build -buildvcs=false -o "$work/go-provider" ./cmd/go-provider
    go build -buildvcs=false -o "$work/go-lift" ./cmd/go-execution-lift
    go build -buildvcs=false -o "$work/pure-lower" ./cmd/pure-wasm-lower
)
(
    cd "$pulp_repo"
    go test -buildvcs=false ./cmd/pulp-seme-function-proof
    go build -buildvcs=false -o "$work/pulp-function" ./cmd/pulp-seme-function-proof
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

# Mixed Boolean/i64 parameters and Boolean result.
cp -R "$repo/fixtures/go-execution-v12" "$work/bool-project"
(cd "$work/bool-project" && go test -buildvcs=false ./...)
"$work/go-provider" ingest --project "$work/bool-project" \
    --module "$repo/modules/provider/v1/module.g1" --out "$work/bool-import"
"$work/go-lift" --project "$work/bool-project" \
    --manifest "$work/bool-import/manifest.json" --module "$execution/module.g1" \
    --function Allowed --profile structured-v8 --out "$work/bool-program.g1"
"$k0" "$g1" "$work/bool-program.g1" "$work/bool-program.seme"
"$k0" "$kernel" "$work/bool-program.seme"
"$k0" "$foundation" "$work/bool-program.seme"
"$work/pure-lower" "$work/bool-program.seme" "$work/bool-a.wasm" "$work/bool-a.json"
"$work/pure-lower" "$work/bool-program.seme" "$work/bool-b.wasm" "$work/bool-b.json"
cmp "$work/bool-a.wasm" "$work/bool-b.wasm"
cmp "$work/bool-a.json" "$work/bool-b.json"
cmp "$target/pure-function.wasm" "$work/bool-a.wasm"
cmp "$target/abi.json" "$work/bool-a.json"
(cd "$target" && sha256sum -c pure-function.wasm.sha256)
rg -q '"request_size": 17' "$work/bool-a.json"
rg -q '"response_size": 1' "$work/bool-a.json"
rg -q '"parameters": \[' "$work/bool-a.json"
rg -q '"result": "bool"' "$work/bool-a.json"

node "$repo/reference/js/pure-function-runner.mjs" "$work/bool-a.wasm" \
    0104000000000000000500000000000000 \
    0004000000000000000500000000000000 \
    0106000000000000000500000000000000 \
    01 \
    0204000000000000000500000000000000 > "$work/standalone-bool.log"
rg -q '"status":0,"response":"01"' "$work/standalone-bool.log"
test "$(rg -c '"status":0,"response":"00"' "$work/standalone-bool.log")" -eq 2
rg -q '"request":"01","status":2' "$work/standalone-bool.log"
rg -q '"request":"02.*","status":3' "$work/standalone-bool.log"

"$work/pulp-function" -manifest "$target/pulp.cell.toml" \
    -provider seme.function-v1 \
    -request 0104000000000000000500000000000000 \
    -request 0004000000000000000500000000000000 \
    -request 0106000000000000000500000000000000 > "$work/pulp-valid.log" 2>&1
rg -q '"response":"01"' "$work/pulp-valid.log"
test "$(rg -c '"response":"00"' "$work/pulp-valid.log")" -eq 2
expect_status 1 "$work/pulp-function" -manifest "$target/pulp.cell.toml" \
    -provider seme.function-v1 -request 01
expect_status 1 "$work/pulp-function" -manifest "$target/pulp.cell.toml" \
    -provider seme.function-v1 -request 0204000000000000000500000000000000

# Ordered i64 parameters and i64 result use the same graph-driven backend.
cp -R "$repo/fixtures/go-execution-v10" "$work/i64-project"
"$work/go-provider" ingest --project "$work/i64-project" \
    --module "$repo/modules/provider/v1/module.g1" --out "$work/i64-import"
"$work/go-lift" --project "$work/i64-project" \
    --manifest "$work/i64-import/manifest.json" --module "$execution/module.g1" \
    --function SubtractNested --profile structured-v8 --out "$work/i64-program.g1"
"$k0" "$g1" "$work/i64-program.g1" "$work/i64-program.seme"
"$work/pure-lower" "$work/i64-program.seme" "$work/i64.wasm" "$work/i64.json"
rg -q '"request_size": 24' "$work/i64.json"
rg -q '"response_size": 8' "$work/i64.json"
rg -q '"result": "i64"' "$work/i64.json"
node "$repo/reference/js/pure-function-runner.mjs" "$work/i64.wasm" \
    140000000000000003000000000000000400000000000000 \
    030000000000000014000000000000000400000000000000 \
    000000000000008001000000000000000000000000000000 > "$work/standalone-i64.log"
rg -q '"response":"0900000000000000"' "$work/standalone-i64.log"
rg -q '"response":"e7ffffffffffffff"' "$work/standalone-i64.log"
rg -q '"response":"ffffffffffffff7f"' "$work/standalone-i64.log"

version=2
while [ "$version" -le 11 ]; do
    (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/module-v$version.g1"
    cmp "$repo/modules/execution/v$version/module.g1" "$work/module-v$version.g1"
    version=$((version + 1))
done

echo "Core Execution v12: generic pure-function ABI ran standalone and through Pulp"
