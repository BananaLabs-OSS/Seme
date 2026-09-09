#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-composite-v32.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" \
  "$repo/conformance/uab-v1/composite-option-result.g1" "$work/program.seme"

if [ -n "${SEME_COMPOSITE_WASM_LOWER:-}" ]; then
  cp "$SEME_COMPOSITE_WASM_LOWER" "$work/lower"
else
  (cd "$repo/reference/go" && \
    go test -buildvcs=false ./wasmtarget && \
    go build -buildvcs=false -o "$work/lower" ./cmd/composite-wasm-lower)
fi
"$work/lower" "$work/program.seme" "$work/first.wasm" "$work/first.json"
"$work/lower" "$work/program.seme" "$work/second.wasm" "$work/second.json"
cmp "$work/first.wasm" "$work/second.wasm"
cmp "$work/first.json" "$work/second.json"
rg -q '"contract": "seme.pure-composite-abi/v1"' "$work/first.json"
rg -q '"provider": "seme.function-composite-v1"' "$work/first.json"
rg -q '"type": "option\\u003cresult\\u003cbytes,text\\u003e\\u003e"' "$work/first.json"
rg -q '"request_fixed_size": 10' "$work/first.json"
rg -q '"response_fixed_size": 1' "$work/first.json"
node "$repo/reference/js/composite-function-runner.mjs" "$work/first.wasm"

if [ ! -f "$pulp_repo/go.mod" ]; then
  echo "Pulp checkout unavailable at $pulp_repo" >&2
  exit 65
fi
if ! git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"; then
  echo "Pulp proof commit unavailable: $pulp_commit" >&2
  exit 65
fi
mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-composite-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/first.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-composite-v1 \
  -request 00000000000000000000 \
  -request 01000a000000020000006f6b \
  -request 01010a00000003000000626164 > "$work/pulp.log" 2>&1
rg -q '"request":"00000000000000000000","response":"00"' "$work/pulp.log"
rg -q '"request":"01000a000000020000006f6b","response":"01"' "$work/pulp.log"
rg -q '"request":"01010a00000003000000626164","response":"01"' "$work/pulp.log"

echo "Composite runtime v32: canonical fixture lowered deterministically and executed standalone and through pinned Pulp"
