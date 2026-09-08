#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-provider.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

(cd "$repo/reference/js" && npm test)
(
    cd "$repo/reference/go"
    go build -buildvcs=false -o "$work/session-proof" ./cmd/go-session-proof
    go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower
)

mkdir "$work/go"
printf '%s\n' \
    'package cross' \
    'func Join(left, right string) string {' \
    '    if left == "" { return right }' \
    '    return left + "λ" + right' \
    '}' > "$work/go/text.go"
printf '%s\n' \
    '/**' \
    ' * @param {string} left' \
    ' * @param {string} right' \
    ' * @returns {string}' \
    ' */' \
    'export function Join(left, right) {' \
    '  if (left === "") return right;' \
    '  return left + "λ" + right;' \
    '}' > "$work/text.mjs"

"$work/session-proof" --module "$repo/modules/execution/v14/module.g1" \
    --project "$work/go" --package example.test/cross-text --revision 1 \
    --out "$work/go.g1"
node "$repo/reference/js/javascript-provider-cli.mjs" \
    --source "$work/text.mjs" --module "$repo/modules/execution/v14/module.g1" \
    --package example.test/cross-text --revision 1 --out "$work/js.g1"
node "$repo/reference/js/javascript-provider-cli.mjs" \
    --source "$work/text.mjs" --module "$repo/modules/execution/v14/module.g1" \
    --package example.test/cross-text --revision 1 --out "$work/js-second.g1"
cmp "$work/js.g1" "$work/js-second.g1"
node "$repo/reference/js/javascript-native-runner.mjs" "file://$work/text.mjs" 雪 🦀 \
    > "$work/native-js.log"
rg -q '"result":"雪λ🦀"' "$work/native-js.log"

for language in go js; do
    "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" \
        "$work/$language.g1" "$work/$language.seme"
    "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" \
        "$work/$language.seme"
    "$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" \
        "$work/$language.seme"
done
if ! cmp "$work/go.seme" "$work/js.seme"; then
    diff -u "$work/go.g1" "$work/js.g1"
    exit 1
fi

"$work/lower" "$work/js.seme" "$work/js.wasm" "$work/js-abi.json"
rg -q '"contract": "seme.pure-abi/v2"' "$work/js-abi.json"
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/js.wasm" \
    10000000030000001300000004000000e99baaf09fa680 e99baacebbf09fa680

echo "JavaScript provider v1: native syntax and Go lifted to identical canonical Seme and executed"
