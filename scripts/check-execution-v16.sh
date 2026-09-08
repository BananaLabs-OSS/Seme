#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v16.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"
export XDG_CACHE_HOME

(
    cd "$repo/reference/go"
    go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget
    go run -buildvcs=false ./cmd/execution-module-v16 > "$work/module.g1"
    go build -buildvcs=false -o "$work/session-proof" ./cmd/go-session-proof
    go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower
)
cmp "$repo/modules/execution/v16/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v16" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v16/module.seme" "$work/module.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/module.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/module.seme"

(cd "$repo/fixtures/go-execution-v16" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test)
"$work/session-proof" --module "$repo/modules/execution/v16/module.g1" \
    --project "$repo/fixtures/go-execution-v16" --package example.com/seme-call-proof \
    --entry Render --revision 1 --out "$work/go.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/go.g1" "$work/go.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/go.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/go.seme"
"$work/lower" "$work/go.seme" "$work/go.wasm" "$work/go-abi.json"
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/go.wasm" \
    10000000030000001300000004000000e99baaf09fa680 5b5be99baa5d5bf09fa6805d5d

mkdir "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-function" ./cmd/pulp-seme-function-proof)
mkdir "$work/cell"
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/cell/pulp.cell.toml"
cp "$work/go.wasm" "$work/cell/pure-function.wasm"
"$work/pulp-function" -manifest "$work/cell/pulp.cell.toml" -provider seme.function-v2 \
    -request 10000000030000001300000004000000e99baaf09fa680 > "$work/pulp.log" 2>&1
rg -q '"response":"5b5be99baa5d5bf09fa6805d5d"' "$work/pulp.log"

printf '%s\n' \
    '/** @param {string} value @returns {string} */' \
    'function decorate(value) { const prefix = "[" + value; return prefix + "]"; }' \
    '/** @param {string} left @param {string} right @returns {string} */' \
    'function combine(left, right) { return decorate(left) + decorate(right); }' \
    '/** @param {string} left @param {string} right @returns {string} */' \
    'export function Render(left, right) { const joined = combine(left, right); return decorate(joined); }' \
    > "$work/program.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" \
    --source "$work/program.mjs" --module "$repo/modules/execution/v16/module.g1" \
    --package example.com/seme-call-proof --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/go.seme" "$work/js.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/go.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" \
    --source "$work/projected.mjs" --module "$repo/modules/execution/v16/module.g1" \
    --package example.com/seme-call-proof --revision 1 --out "$work/relifted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/go.seme" "$work/relifted.seme"
node "$repo/reference/js/javascript-native-runner.mjs" "file://$work/projected.mjs" 雪 🦀 Render > "$work/native-js.log"
rg -q '"result":"\[\[雪\]\[🦀\]\]"' "$work/native-js.log"

mkdir "$work/recursive"
printf '%s\n' 'package recursive' \
    'func loop(value string) string { return loop(value) }' \
    'func Render(value string) string { return loop(value) }' > "$work/recursive/recursive.go"
"$work/session-proof" --module "$repo/modules/execution/v16/module.g1" \
    --project "$work/recursive" --package example.test/recursive --entry Render \
    --revision 1 --out "$work/recursive.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/recursive.g1" "$work/recursive.seme"
if "$work/lower" "$work/recursive.seme" "$work/recursive.wasm" "$work/recursive-abi.json" 2> "$work/recursive.err"; then
    echo "recursive call graph unexpectedly lowered" >&2
    exit 1
fi
rg -q 'wasm.pure_recursive_call' "$work/recursive.err"

version=2
while [ "$version" -le 15 ]; do
    (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/module-v$version.g1"
    cmp "$repo/modules/execution/v$version/module.g1" "$work/module-v$version.g1"
    version=$((version + 1))
done

echo "Core Execution v16: closed Go/JavaScript call graph ran natively, standalone, and through Pulp"
