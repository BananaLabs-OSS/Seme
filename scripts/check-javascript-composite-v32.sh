#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-composite.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME
GOCACHE="$work/go-cache"; export GOCACHE

node "$repo/reference/js/javascript-provider-cli.mjs" \
  --source "$repo/fixtures/javascript-uab-02/composite.js" \
  --module "$repo/modules/execution/v32/module.g1" \
  --package example.test/javascript-uab-02 --revision 1 --out "$work/source.g1"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/source.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" \
  --source "$work/projected.mjs" --module "$repo/modules/execution/v32/module.g1" \
  --package example.test/javascript-uab-02 --revision 1 --out "$work/relifted.g1"
cmp "$work/source.g1" "$work/relifted.g1"
node "$repo/reference/js/javascript-composite-native-runner.mjs"

"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/source.g1" "$work/program.seme"
if [ -n "${SEME_COMPOSITE_WASM_LOWER:-}" ]; then
  cp "$SEME_COMPOSITE_WASM_LOWER" "$work/lower"
else
  (cd "$repo/reference/go" && go build -buildvcs=false -o "$work/lower" ./cmd/composite-wasm-lower)
fi
"$work/lower" "$work/program.seme" "$work/program.wasm" "$work/abi.json"
node "$repo/reference/js/composite-function-runner.mjs" "$work/program.wasm"

pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
if [ ! -f "$pulp_repo/go.mod" ] || ! git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"; then
  echo "Pinned Pulp proof checkout unavailable" >&2
  exit 65
fi
mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-composite-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/program.wasm" "$work/pulp/pure-function.wasm"
(cd "$work/pulp" && "$work/pulp-runner" -manifest pulp.cell.toml -provider seme.function-composite-v1 \
  -request 00000000000000000000 \
  -request 01000a000000020000006f6b \
  -request 01010a00000003000000626164) > "$work/pulp.log" 2>&1 || { cat "$work/pulp.log" >&2; exit 1; }
rg -q '"request":"00000000000000000000","response":"00"' "$work/pulp.log"
rg -q '"request":"01000a000000020000006f6b","response":"01"' "$work/pulp.log"
rg -q '"request":"01010a00000003000000626164","response":"01"' "$work/pulp.log"

echo "JavaScript composite v32: direct lift, native parity, projection/re-lift, Wasm, and pinned Pulp parity pass"
