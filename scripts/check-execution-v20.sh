#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v20.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
 go run -buildvcs=false ./cmd/execution-module-v20 > "$work/module.g1" && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
cmp "$repo/modules/execution/v20/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v20" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v20/module.seme" "$work/module.seme"

(cd "$repo/fixtures/go-execution-v20" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test)
"$work/session" --module "$repo/modules/execution/v20/module.g1" \
  --project "$repo/fixtures/go-execution-v20" --package example.com/seme-effect-proof \
  --entry Observe --revision 1 --out "$work/go.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/go.g1" "$work/go.seme"
"$work/lower" "$work/go.seme" "$work/effect.wasm" "$work/effect-abi.json"
rg -q '"fidelity": "adapted"' "$work/effect-abi.json"
rg -q '"observability.log"' "$work/effect-abi.json"
node "$repo/reference/js/pure-function-effect-runner.mjs" "$work/effect.wasm" 0100 00 true,false granted
node "$repo/reference/js/pure-function-effect-runner.mjs" "$work/effect.wasm" 0100 00 true,false denied

printf '%s\n' '/** @param {boolean} first @param {boolean} second @returns {boolean} */' \
  'export function Observe(first, second) {' '  console.log(first);' '  console.log(second);' '  return second;' '}' > "$work/program.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/program.mjs" \
  --module "$repo/modules/execution/v20/module.g1" --package example.com/seme-effect-proof \
  --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/go.seme" "$work/js.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/go.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v20/module.g1" --package example.com/seme-effect-proof \
  --revision 1 --out "$work/relifted.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/relifted.g1" "$work/relifted.seme"
cmp "$work/go.seme" "$work/relifted.seme"
node "$repo/reference/js/javascript-effect-native-runner.mjs" "file://$work/projected.mjs" true false > "$work/native.log"
rg -q '"result":false,"events":\[true,false\]' "$work/native.log"

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
cp "$repo/targets/wasm/pulp-effect-v1/capability.go" "$work/pulp-pinned/cmd/pulp-seme-function-proof/capability.go"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-effect-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$repo/targets/wasm/pulp-effect-v1/pulp.denied.cell.toml" "$work/pulp/pulp.denied.cell.toml"
cp "$work/effect.wasm" "$work/pulp/effect-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 \
  -request 0100 > "$work/pulp-allowed.log" 2>&1
test "$(rg -c '^\[observability.log\]' "$work/pulp-allowed.log")" -eq 2
first_line=$(rg '^\[observability.log\]' "$work/pulp-allowed.log" | sed -n '1p')
second_line=$(rg '^\[observability.log\]' "$work/pulp-allowed.log" | sed -n '2p')
printf '%s' "$first_line" | rg -q 'value=true'
printf '%s' "$second_line" | rg -q 'value=false'
rg -q '"response":"00"' "$work/pulp-allowed.log"
if "$work/pulp-runner" -manifest "$work/pulp/pulp.denied.cell.toml" -provider seme.function-v1 \
  -request 0100 > "$work/pulp-denied.log" 2>&1; then
  echo "denied effect cell succeeded" >&2
  exit 1
fi
if rg -q '^\[observability.log\]' "$work/pulp-denied.log"; then
  echo "denied effect cell emitted an observation" >&2
  exit 1
fi

printf '%s\n' '/** @param {boolean} value @returns {boolean} */' \
  'export function Invalid(value) { console.error(value); return value; }' > "$work/invalid.mjs"
if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/invalid.mjs" \
  --module "$repo/modules/execution/v20/module.g1" --package example.com/seme-invalid-effect \
  --revision 1 --out "$work/invalid.g1" > "$work/invalid.log" 2>&1; then
  echo "undeclared effect adapter was accepted" >&2
  exit 1
fi
rg -q 'javascript.unsupported_effect' "$work/invalid.log"

version=2
while [ "$version" -le 19 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo "Core Execution v20: ordered Go/JavaScript effects ran with granted capability and trapped without it"
