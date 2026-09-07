#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-canonical-check.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

k0=$repo/bootstrap/seme-k0-linux-amd64
execution=$repo/modules/execution/v12/module.g1

(cd "$repo/reference/go" && \
    go build -buildvcs=false -o "$work/go-provider" ./cmd/go-provider && \
    go build -buildvcs=false -o "$work/go-lift" ./cmd/go-execution-lift)
cp -R "$repo/fixtures/go-execution-v12" "$work/source"
"$work/go-provider" ingest --project "$work/source" \
    --module "$repo/modules/provider/v1/module.g1" --out "$work/import"
"$work/go-lift" --project "$work/source" \
    --manifest "$work/import/manifest.json" --module "$execution" \
    --function Allowed --profile structured-v8 --out "$work/program.g1"
"$k0" "$repo/compiler/g1-compiler.k0" "$work/program.g1" "$work/program.seme"

# The builder receives canonical bytes and pins only. Removing every source-side
# input before invoking it makes an accidental provider dependency observable.
rm -rf "$work/source" "$work/import" "$work/program.g1"
revision=720ca2132730f5338325ec050ce12593
program_sha256=8a640d18ec4e550c03df2059bc8d847aafd0a6302d1c7bd082536afb6047adca
"$repo/scripts/build-certified-canonical-v1.sh" "$work/program.seme" \
    "$revision" "$program_sha256" "$work/output.wasm" "$work/output.json"
cmp "$repo/targets/wasm/pulp-function-v1/pure-function.wasm" "$work/output.wasm"
cmp "$repo/targets/wasm/pulp-function-v1/abi.json" "$work/output.json"

printf 'artifact-before\n' > "$work/stale.wasm"
printf 'evidence-before\n' > "$work/stale.json"
cp "$work/stale.wasm" "$work/stale-wasm.expected"
cp "$work/stale.json" "$work/stale-json.expected"
set +e
"$repo/scripts/build-certified-canonical-v1.sh" "$work/program.seme" \
    "$revision" 0000000000000000000000000000000000000000000000000000000000000000 \
    "$work/stale.wasm" "$work/stale.json" > "$work/stale.out" 2> "$work/stale.err"
status=$?
set -e
[ "$status" -eq 65 ]
grep -q 'canonical digest mismatch' "$work/stale.err"
cmp "$work/stale-wasm.expected" "$work/stale.wasm"
cmp "$work/stale-json.expected" "$work/stale.json"

echo "Certified canonical build v1: pinned canonical bytes built without source authority; stale input preserved outputs"

