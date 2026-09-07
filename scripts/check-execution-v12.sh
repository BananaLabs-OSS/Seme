#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
pulp_tree=a11daea8a78b5430fc7429b30126a6d0f989ad29
pulp_runner_sha256=5dca76d9ba6a7e7d9293e1a8357359036ce7fd475f91fecfb508495487fd77f1
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
git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
test "$(git -C "$pulp_repo" rev-parse "$pulp_commit^{tree}")" = "$pulp_tree"
git -C "$pulp_repo" merge-base --is-ancestor "$pulp_commit" HEAD
mkdir "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
test "$(sha256sum "$work/pulp-pinned/cmd/pulp-seme-function-proof/main.go" | cut -d ' ' -f 1)" = "$pulp_runner_sha256"

(
    cd "$repo/reference/go"
    go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget
    go run -buildvcs=false ./cmd/execution-module-v12 > "$work/module.g1"
    go build -buildvcs=false -o "$work/go-provider" ./cmd/go-provider
    go build -buildvcs=false -o "$work/go-lift" ./cmd/go-execution-lift
    go build -buildvcs=false -o "$work/pure-lower" ./cmd/pure-wasm-lower
)
sh "$repo/scripts/check-core-v12-certificate.sh"
(
    cd "$work/pulp-pinned"
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
rg -q '"contract": "seme.pure-abi/v1"' "$work/bool-a.json"
rg -q '"provider": "seme.function-v1"' "$work/bool-a.json"
rg -q '"target": "wasm32-pulp-reactor-v1"' "$work/bool-a.json"
rg -q '"fidelity": "exact"' "$work/bool-a.json"
rg -q '"program_sha256": "[0-9a-f]{64}"' "$work/bool-a.json"
rg -q '"artifact_sha256": "[0-9a-f]{64}"' "$work/bool-a.json"
rg -q '"offset": 0' "$work/bool-a.json"
rg -q '"encoding": "canonical-u8-0-or-1"' "$work/bool-a.json"
rg -q '"type": "bool"' "$work/bool-a.json"

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
node "$repo/reference/js/pure-function-hardening.mjs" "$work/bool-a.wasm" \
    0104000000000000000500000000000000

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
rg -q '"type": "i64"' "$work/i64.json"
rg -q '"encoding": "little-endian-twos-complement-i64-modular"' "$work/i64.json"
node "$repo/reference/js/pure-function-runner.mjs" "$work/i64.wasm" \
    140000000000000003000000000000000400000000000000 \
    030000000000000014000000000000000400000000000000 \
    000000000000008001000000000000000000000000000000 > "$work/standalone-i64.log"
rg -q '"response":"0900000000000000"' "$work/standalone-i64.log"
rg -q '"response":"e7ffffffffffffff"' "$work/standalone-i64.log"
rg -q '"response":"ffffffffffffff7f"' "$work/standalone-i64.log"

mkdir "$work/i64-cell"
cp "$target/pulp.cell.toml" "$work/i64-cell/pulp.cell.toml"
cp "$work/i64.wasm" "$work/i64-cell/pure-function.wasm"
"$work/pulp-function" -manifest "$work/i64-cell/pulp.cell.toml" \
    -provider seme.function-v1 \
    -request 140000000000000003000000000000000400000000000000 \
    -request 030000000000000014000000000000000400000000000000 \
    -request 000000000000008001000000000000000000000000000000 > "$work/pulp-i64.log" 2>&1
rg -q '"response":"0900000000000000"' "$work/pulp-i64.log"
rg -q '"response":"e7ffffffffffffff"' "$work/pulp-i64.log"
rg -q '"response":"ffffffffffffff7f"' "$work/pulp-i64.log"

version=2
while [ "$version" -le 11 ]; do
    (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/module-v$version.g1"
    cmp "$repo/modules/execution/v$version/module.g1" "$work/module-v$version.g1"
    version=$((version + 1))
done

echo "Core Execution v12: generic pure-function ABI ran standalone and through Pulp"
