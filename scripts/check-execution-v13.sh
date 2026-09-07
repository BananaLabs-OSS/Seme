#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
pulp_tree=a11daea8a78b5430fc7429b30126a6d0f989ad29
pulp_runner_sha256=5dca76d9ba6a7e7d9293e1a8357359036ce7fd475f91fecfb508495487fd77f1
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v13.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

k0="$repo/bootstrap/seme-k0-linux-amd64"
g1="$repo/compiler/g1-compiler.k0"
kernel="$repo/compiler/kernel-wire-validator.k0"
foundation="$repo/modules/foundation/v1/validator.k0"
execution="$repo/modules/execution/v13"

git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
test "$(git -C "$pulp_repo" rev-parse "$pulp_commit^{tree}")" = "$pulp_tree"
git -C "$pulp_repo" merge-base --is-ancestor "$pulp_commit" HEAD
mkdir "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
test "$(sha256sum "$work/pulp-pinned/cmd/pulp-seme-function-proof/main.go" | cut -d ' ' -f 1)" = "$pulp_runner_sha256"

(
    cd "$repo/reference/go"
    go test -buildvcs=false ./...
    go run -buildvcs=false ./cmd/execution-module-v13 > "$work/module.g1"
    go build -buildvcs=false -o "$work/go-provider" ./cmd/go-provider
    go build -buildvcs=false -o "$work/go-lift" ./cmd/go-execution-lift
    go build -buildvcs=false -o "$work/pure-lower" ./cmd/pure-wasm-lower
)
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

cp -R "$repo/fixtures/go-execution-v13" "$work/project"
(cd "$work/project" && go test -buildvcs=false ./...)
"$work/go-provider" ingest --project "$work/project" \
    --module "$repo/modules/provider/v1/module.g1" --out "$work/import"
"$work/go-lift" --project "$work/project" \
    --manifest "$work/import/manifest.json" --module "$execution/module.g1" \
    --function Decide --profile control-v13 --out "$work/program-a.g1"
"$work/go-lift" --project "$work/project" \
    --manifest "$work/import/manifest.json" --module "$execution/module.g1" \
    --function Decide --profile control-v13 --out "$work/program-b.g1"
cmp "$work/program-a.g1" "$work/program-b.g1"
"$k0" "$g1" "$work/program-a.g1" "$work/program.seme"
"$k0" "$kernel" "$work/program.seme"
"$k0" "$foundation" "$work/program.seme"
"$work/pure-lower" "$work/program.seme" "$work/function-a.wasm" "$work/abi-a.json"
"$work/pure-lower" "$work/program.seme" "$work/function-b.wasm" "$work/abi-b.json"
cmp "$work/function-a.wasm" "$work/function-b.wasm"
cmp "$work/abi-a.json" "$work/abi-b.json"

node "$repo/reference/js/pure-function-runner.mjs" "$work/function-a.wasm" \
    0104000000000000000500000000000000 \
    0106000000000000000500000000000000 \
    0004000000000000000500000000000000 > "$work/standalone.log"
rg -q '"request":"0104.*"status":0,"response":"01"' "$work/standalone.log"
test "$(rg -c '"status":0,"response":"00"' "$work/standalone.log")" -eq 2

mkdir "$work/cell"
cp "$repo/targets/wasm/pulp-function-v1/pulp.cell.toml" "$work/cell/pulp.cell.toml"
cp "$work/function-a.wasm" "$work/cell/pure-function.wasm"
"$work/pulp-function" -manifest "$work/cell/pulp.cell.toml" -provider seme.function-v1 \
    -request 0104000000000000000500000000000000 \
    -request 0106000000000000000500000000000000 \
    -request 0004000000000000000500000000000000 > "$work/pulp.log" 2>&1
rg -q '"response":"01"' "$work/pulp.log"
test "$(rg -c '"response":"00"' "$work/pulp.log")" -eq 2

version=2
while [ "$version" -le 12 ]; do
    (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/module-v$version.g1"
    cmp "$repo/modules/execution/v$version/module.g1" "$work/module-v$version.g1"
    version=$((version + 1))
done

echo "Core Execution v13: generic branching and text ran standalone and through Pulp"
