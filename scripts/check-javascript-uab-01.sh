#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-uab-01.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/js" && npm test)
if [ -n "${SEME_PURE_WASM_LOWER:-}" ]; then
  cp "$SEME_PURE_WASM_LOWER" "$work/lower"
else
  (cd "$repo/reference/go" && go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
fi

cd "$repo"
node reference/js/javascript-package-provider-cli.mjs \
  --files fixtures/javascript-uab-01/application.js,fixtures/javascript-uab-01/math/sum.js \
  --module modules/execution/v30/module.g1 \
  --package example.test/multi-source --entry Run --revision 1 \
  --out "$work/package.g1"
bootstrap/seme-k0-linux-amd64 compiler/g1-compiler.k0 "$work/package.g1" "$work/package.seme"
node reference/js/javascript-projector-cli.mjs "$work/package.g1" "$work/projected.mjs"
node --check "$work/projected.mjs"
node reference/js/javascript-provider-cli.mjs \
  --source "$work/projected.mjs" --module modules/execution/v30/module.g1 \
  --package example.test/multi-source --entry Run --revision 1 --out "$work/relift.g1"
bootstrap/seme-k0-linux-amd64 compiler/g1-compiler.k0 "$work/relift.g1" "$work/relift.seme"
cmp "$work/package.seme" "$work/relift.seme"

"$work/lower" "$work/package.seme" "$work/package.wasm" "$work/package-abi.json"
"$work/lower" "$work/package.seme" "$work/package-second.wasm" "$work/package-second-abi.json"
cmp "$work/package.wasm" "$work/package-second.wasm"
cmp "$work/package-abi.json" "$work/package-second-abi.json"
rg -q '"contract": "seme.pure-abi/v1"' "$work/package-abi.json"
rg -q '"request_size": 16' "$work/package-abi.json"
rg -q '"response_size": 8' "$work/package-abi.json"

# Run(7, 5) = 12 and Run(-7, 5) = -2 using little-endian signed i64 words.
node reference/js/pure-function-runner.mjs "$work/package.wasm" \
  07000000000000000500000000000000 \
  f9ffffffffffffff0500000000000000 > "$work/standalone.log"
rg -q '"response":"0c00000000000000"' "$work/standalone.log"
rg -q '"response":"feffffffffffffff"' "$work/standalone.log"

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp targets/wasm/pulp-function-v1/pulp.cell.toml "$work/pulp/pulp.cell.toml"
cp "$work/package.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" \
  -provider seme.function-v1 -request 07000000000000000500000000000000 > "$work/pulp.log" 2>&1
rg -q '"response":"0c00000000000000"' "$work/pulp.log"

echo "JavaScript UAB-01: multi-source lift, native parity, Wasm/Pulp parity, projection/re-lift, and rejection pass"
