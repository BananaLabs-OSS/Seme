#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v15.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"
export XDG_CACHE_HOME

(
    cd "$repo/reference/go"
    go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget
    go run -buildvcs=false ./cmd/execution-module-v15 > "$work/module.g1"
    go build -buildvcs=false -o "$work/session-proof" ./cmd/go-session-proof
    go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower
)
cmp "$repo/modules/execution/v15/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v15" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" \
    "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v15/module.seme" "$work/module.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/module.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/module.seme"

(cd "$repo/fixtures/go-execution-v15" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test)
"$work/session-proof" --module "$repo/modules/execution/v15/module.g1" \
    --project "$repo/fixtures/go-execution-v15" --package example.com/seme-local-proof \
    --revision 1 --out "$work/program.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" \
    "$work/program.g1" "$work/program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/program.seme"
"$work/lower" "$work/program.seme" "$work/program.wasm" "$work/abi.json"
node "$repo/reference/js/pure-function-runner.mjs" "$work/program.wasm" \
    04000000000000000500000000000000 > "$work/runtime.log"
rg -q '"status":0,"response":"0e00000000000000"' "$work/runtime.log"

mkdir "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-function" ./cmd/pulp-seme-function-proof)
mkdir "$work/cell"
cp "$repo/targets/wasm/pulp-function-v1/pulp.cell.toml" "$work/cell/pulp.cell.toml"
cp "$work/program.wasm" "$work/cell/pure-function.wasm"
"$work/pulp-function" -manifest "$work/cell/pulp.cell.toml" -provider seme.function-v1 \
    -request 04000000000000000500000000000000 > "$work/pulp.log" 2>&1
rg -q '"response":"0e00000000000000"' "$work/pulp.log"

mkdir "$work/go-text"
printf '%s\n' \
    'package localtext' \
    'func Join(left, right string) string {' \
    '    middle := left + "λ"' \
    '    complete := middle + right' \
    '    return complete' \
    '}' > "$work/go-text/text.go"
printf '%s\n' \
    '/**' \
    ' * @param {string} left' \
    ' * @param {string} right' \
    ' * @returns {string}' \
    ' */' \
    'export function Join(left, right) {' \
    '  const middle = left + "λ";' \
    '  const complete = middle + right;' \
    '  return complete;' \
    '}' > "$work/text.mjs"
"$work/session-proof" --module "$repo/modules/execution/v15/module.g1" \
    --project "$work/go-text" --package example.test/cross-local-text --revision 1 \
    --out "$work/go-text.g1"
node "$repo/reference/js/javascript-provider-cli.mjs" \
    --source "$work/text.mjs" --module "$repo/modules/execution/v15/module.g1" \
    --package example.test/cross-local-text --revision 1 --out "$work/js-text.g1"
for language in go-text js-text; do
    "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" \
        "$work/$language.g1" "$work/$language.seme"
done
cmp "$work/go-text.seme" "$work/js-text.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/go-text.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" \
    --source "$work/projected.mjs" --module "$repo/modules/execution/v15/module.g1" \
    --package example.test/cross-local-text --revision 1 --out "$work/relifted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" \
    "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/go-text.seme" "$work/relifted.seme"
"$work/lower" "$work/js-text.seme" "$work/js-text.wasm" "$work/js-text-abi.json"
node "$repo/reference/js/javascript-native-runner.mjs" "file://$work/projected.mjs" 雪 🦀 \
    > "$work/projected.log"
rg -q '"result":"雪λ🦀"' "$work/projected.log"
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/js-text.wasm" \
    10000000030000001300000004000000e99baaf09fa680 e99baacebbf09fa680

version=2
while [ "$version" -le 14 ]; do
    (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/module-v$version.g1"
    cmp "$repo/modules/execution/v$version/module.g1" "$work/module-v$version.g1"
    version=$((version + 1))
done

echo "Core Execution v15: immutable lexical locals ran natively, standalone, and through Pulp"
