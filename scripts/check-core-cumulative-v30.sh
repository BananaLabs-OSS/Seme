#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-core-cumulative.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

# Preserve the complete v2-v30 checkpoint evidence before testing composition.
"$repo/scripts/check-execution-v30.sh"

(cd "$repo/reference/go" && \
  go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
  go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
  go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
(cd "$repo/fixtures/go-cumulative-proof-a" && go test -buildvcs=false ./...)
(cd "$repo/fixtures/go-cumulative-proof-b" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && \
  npm test && \
  node cumulative-state-flow-native-runner.mjs && \
  node cumulative-text-collection-native-runner.mjs)

prove_language_pair() {
  name=$1
  fixture=$2
  package_path=$3
  entry=$4

  "$work/session" --module "$repo/modules/execution/v30/module.g1" \
    --project "$repo/$fixture" --package "$package_path" \
    --entry "$entry" --revision 1 --out "$work/$name-go.g1"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" \
    "$work/$name-go.g1" "$work/$name-go.seme"

  node "$repo/reference/js/javascript-projector-cli.mjs" \
    "$work/$name-go.g1" "$work/$name-projected.mjs"
  node --check "$work/$name-projected.mjs"
  node "$repo/reference/js/javascript-provider-cli.mjs" \
    --source "$work/$name-projected.mjs" \
    --module "$repo/modules/execution/v30/module.g1" \
    --package "$package_path" --entry "$entry" --revision 1 \
    --out "$work/$name-js.g1"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" \
    "$work/$name-js.g1" "$work/$name-js.seme"
  cmp "$work/$name-go.seme" "$work/$name-js.seme"

  "$work/lower" "$work/$name-go.seme" "$work/$name.wasm" "$work/$name-abi.json"
  "$work/lower" "$work/$name-go.seme" "$work/$name-second.wasm" "$work/$name-second-abi.json"
  cmp "$work/$name.wasm" "$work/$name-second.wasm"
  cmp "$work/$name-abi.json" "$work/$name-second-abi.json"
  rg -q '"contract": "seme.pure-abi/v2"' "$work/$name-abi.json"
  rg -q '"maximum_request_size": 7160' "$work/$name-abi.json"
}

prove_language_pair state \
  fixtures/go-cumulative-proof-a example.test/cumulative-state-flow Run
prove_language_pair text \
  fixtures/go-cumulative-proof-b example.test/cumulative-text-collection Describe

rg -q '"fixed_header_size": 17' "$work/state-abi.json"
rg -q '"response_size": 16' "$work/state-abi.json"
rg -q '"fixed_header_size": 16' "$work/text-abi.json"
node "$repo/reference/js/cumulative-state-flow-runner.mjs" "$work/state.wasm"
node "$repo/reference/js/cumulative-text-collection-runner.mjs" "$work/text.wasm"

# The canonical graph remains structurally valid when the called Sum function
# is removed from Program membership, but the target must refuse to invent an
# executable relationship that the program does not declare.
awk '
  $1 == "fi" && $2 == "00000000000000000000000000009150" && $3 == "li" && $4 == "2" {
    print $1, $2, $3, "1"; in_members = 1; kept = 0; next
  }
  in_members && $1 == "rf" {
    if (kept == 0) { print; kept = 1; next }
    in_members = 0; next
  }
  { print }
' "$work/state-go.g1" > "$work/state-nonmember-call.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" \
  "$work/state-nonmember-call.g1" "$work/state-nonmember-call.seme"
if "$work/lower" "$work/state-nonmember-call.seme" \
  "$work/state-nonmember-call.wasm" "$work/state-nonmember-call.json" >/dev/null 2>&1; then
  echo "nonmember cumulative call was accepted" >&2
  exit 1
fi

# Execute both independently lowered canonical programs through pinned Pulp.
mkdir "$work/pulp-state" "$work/pulp-text" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && \
  go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp-state/pulp.cell.toml"
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp-text/pulp.cell.toml"
cp "$work/state.wasm" "$work/pulp-state/pure-function.wasm"
cp "$work/text.wasm" "$work/pulp-text/pure-function.wasm"

state_request=f9ffffffffffffff110000000300000001030000000000000004000000000000000500000000000000
"$work/pulp-runner" -manifest "$work/pulp-state/pulp.cell.toml" \
  -provider seme.function-v2 -request "$state_request" > "$work/pulp-state.log" 2>&1
rg -q '"response":"05000000000000000500000000000000"' "$work/pulp-state.log"

text_request=1000000003000000130000000200000073756d03000000000000000400000000000000
"$work/pulp-runner" -manifest "$work/pulp-text/pulp.cell.toml" \
  -provider seme.function-v2 -request "$text_request" > "$work/pulp-text.log" 2>&1
rg -q '"response":"73756d3a706f736974697665"' "$work/pulp-text.log"

echo "Core v30 cumulative composition: Go, JavaScript, canonical Seme, Wasm, and Pulp agree"
