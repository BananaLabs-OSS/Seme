#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-session-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

(
    cd "$repo/reference/go"
    go test -buildvcs=false ./goprovider -run 'TestIncrementalSession' -count=1
    go build -buildvcs=false -o "$work/session-proof" ./cmd/go-session-proof
    go build -buildvcs=false -o "$work/pure-lower" ./cmd/pure-wasm-lower
)

"$work/session-proof" \
    --module "$repo/modules/execution/v12/module.g1" \
    --project "$repo/fixtures/go-execution-v12" \
    --package example.com/seme-pure-function-proof \
    --revision 1 --out "$work/session-program.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" \
    "$work/session-program.g1" "$work/session-program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" \
    "$work/session-program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" \
    "$work/session-program.seme"
"$work/pure-lower" "$work/session-program.seme" "$work/session-program.wasm" "$work/session-abi.json"
node "$repo/reference/js/pure-function-runner.mjs" "$work/session-program.wasm" \
    0104000000000000000500000000000000 > "$work/session-run.log"
rg -q '"status":0,"response":"01"' "$work/session-run.log"

mkdir "$work/text-project"
printf '%s\n' 'package text' 'func Join(left, right string) string { return left + "λ" + right }' \
    > "$work/text-project/text.go"
"$work/session-proof" \
    --module "$repo/modules/execution/v14/module.g1" \
    --project "$work/text-project" --package example.test/live-text \
    --revision 1 --out "$work/text-program.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" \
    "$work/text-program.g1" "$work/text-program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" \
    "$work/text-program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" \
    "$work/text-program.seme"
"$work/pure-lower" "$work/text-program.seme" "$work/text-program.wasm" "$work/text-abi.json"
rg -q '"contract": "seme.pure-abi/v2"' "$work/text-abi.json"
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/text-program.wasm" \
    10000000030000001300000004000000e99baaf09fa680 e99baacebbf09fa680 \
    > "$work/text-run.log"
rg -q '"status":0,"response":"e99baacebbf09fa680"' "$work/text-run.log"

# The scalar session proof remains covered by the complete v12 certification,
# Wasm, standalone, and Pulp gate; the text proof above uses the additive v14
# ABI over the same canonical graph path.
sh "$repo/scripts/check-execution-v12.sh"

echo "Go incremental session v1: revision, retention, diagnostics, scalar runtime, and UTF-8 runtime passed"
