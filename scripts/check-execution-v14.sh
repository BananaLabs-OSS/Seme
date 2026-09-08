#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
pulp_tree=a11daea8a78b5430fc7429b30126a6d0f989ad29
pulp_runner_sha256=5dca76d9ba6a7e7d9293e1a8357359036ce7fd475f91fecfb508495487fd77f1
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v14.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"
export XDG_CACHE_HOME

k0="$repo/bootstrap/seme-k0-linux-amd64"
g1="$repo/compiler/g1-compiler.k0"
kernel="$repo/compiler/kernel-wire-validator.k0"
foundation="$repo/modules/foundation/v1/validator.k0"
execution="$repo/modules/execution/v14"

git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
test "$(git -C "$pulp_repo" rev-parse "$pulp_commit^{tree}")" = "$pulp_tree"
git -C "$pulp_repo" merge-base --is-ancestor "$pulp_commit" HEAD
mkdir "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
test "$(sha256sum "$work/pulp-pinned/cmd/pulp-seme-function-proof/main.go" | cut -d ' ' -f 1)" = "$pulp_runner_sha256"

(
    cd "$repo/reference/go"
    go test -buildvcs=false ./...
    go run -buildvcs=false ./cmd/execution-module-v14 > "$work/module.g1"
    go build -buildvcs=false -o "$work/provider" ./cmd/go-provider
    go build -buildvcs=false -o "$work/lift" ./cmd/go-execution-lift
    go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower
)
node --test "$repo/reference/js/string-abi-v2.test.mjs"
(
    cd "$work/pulp-pinned"
    go test -buildvcs=false ./cmd/pulp-seme-function-proof
    go build -buildvcs=false -o "$work/pulp-function" ./cmd/pulp-seme-function-proof
)

cmp "$execution/module.g1" "$work/module.g1"
(cd "$execution" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$k0" "$g1" "$execution/module.g1" "$work/module.seme"
cmp "$execution/module.seme" "$work/module.seme"
"$k0" "$kernel" "$execution/module.seme"
"$k0" "$foundation" "$execution/module.seme"

cp -R "$repo/fixtures/go-execution-v14" "$work/project"
(cd "$work/project" && go test -buildvcs=false ./...)
"$work/provider" ingest --project "$work/project" --module "$repo/modules/provider/v1/module.g1" --out "$work/import"

for function in Join Equal; do
    "$work/lift" --project "$work/project" --manifest "$work/import/manifest.json" \
        --module "$execution/module.g1" --function "$function" --profile structured-v8 --out "$work/$function-a.g1"
    "$work/lift" --project "$work/project" --manifest "$work/import/manifest.json" \
        --module "$execution/module.g1" --function "$function" --profile structured-v8 --out "$work/$function-b.g1"
    cmp "$work/$function-a.g1" "$work/$function-b.g1"
    "$k0" "$g1" "$work/$function-a.g1" "$work/$function.seme"
    "$k0" "$kernel" "$work/$function.seme"
    "$k0" "$foundation" "$work/$function.seme"
    "$work/lower" "$work/$function.seme" "$work/$function.wasm" "$work/$function.json"
    rg -q '"contract": "seme.pure-abi/v2"' "$work/$function.json"
    rg -q '"provider": "seme.function-v2"' "$work/$function.json"
    rg -q '"fixed_header_size": 16' "$work/$function.json"
    rg -q '"maximum_request_size": 7160' "$work/$function.json"
    rg -q '"variable_payload": true' "$work/$function.json"
    rg -q '"encoding": "u32le-offset-u32le-length/utf8-scalar-exact"' "$work/$function.json"
done

node "$repo/reference/js/pure-function-v2-runtime.test.mjs" "$work/Join.wasm" "$work/Equal.wasm"

for function in Join Equal; do
    mkdir "$work/$function-cell"
    cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/$function-cell/pulp.cell.toml"
    cp "$work/$function.wasm" "$work/$function-cell/pure-function.wasm"
done
"$work/pulp-function" -manifest "$work/Join-cell/pulp.cell.toml" -provider seme.function-v2 \
    -request 10000000030000001300000004000000e99baaf09fa680 \
    -request 10000000000000001000000000000000 > "$work/pulp-join.log" 2>&1
rg -q '"response":"e99baaf09fa680"' "$work/pulp-join.log"
rg -q '"response":""' "$work/pulp-join.log"
"$work/pulp-function" -manifest "$work/Equal-cell/pulp.cell.toml" -provider seme.function-v2 \
    -request 10000000030000001300000003000000e99baae99baa > "$work/pulp-equal.log" 2>&1
rg -q '"response":"01"' "$work/pulp-equal.log"

version=2
while [ "$version" -le 13 ]; do
    (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/module-v$version.g1"
    cmp "$repo/modules/execution/v$version/module.g1" "$work/module-v$version.g1"
    version=$((version + 1))
done

echo "Core Execution v14: variable-width UTF-8 ABI ran standalone and through Pulp"
