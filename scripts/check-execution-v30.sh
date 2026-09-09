#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v30.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
 go run -buildvcs=false ./cmd/execution-module-v30 > "$work/module.g1" && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
cmp "$repo/modules/execution/v30/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v30" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v30/module.seme" "$work/module.seme"
(cd "$repo/fixtures/go-execution-v30" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test && node runtime-map-native-runner.mjs)

"$work/session" --module "$repo/modules/execution/v30/module.g1" \
  --project "$repo/fixtures/go-execution-v30" --package example.test/runtime-map \
  --entry Tally --revision 1 --out "$work/go.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/go.g1" "$work/go.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/go.g1" "$work/projected.mjs"
node --check "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v30/module.g1" --package example.test/runtime-map \
  --entry Tally --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/go.seme" "$work/js.seme"

"$work/lower" "$work/go.seme" "$work/tally.wasm" "$work/abi.json"
"$work/lower" "$work/go.seme" "$work/tally-second.wasm" "$work/abi-second.json"
cmp "$work/tally.wasm" "$work/tally-second.wasm"
cmp "$work/abi.json" "$work/abi-second.json"
rg -q '"contract": "seme.pure-abi/v2"' "$work/abi.json"
rg -q '"request_size": 16' "$work/abi.json"
rg -q '"response_size": 8' "$work/abi.json"
rg -q '"maximum_request_size": 7160' "$work/abi.json"
node "$repo/reference/js/runtime-map-runner.mjs" "$work/tally.wasm"

integer=$(awk '$3 == "00000000000000000000000000009010" { print $2; exit }' "$work/go.g1")
boolean=$(awk '$3 == "00000000000000000000000000009020" { print $2; exit }' "$work/go.g1")
map_type=$(awk '$3 == "0000000000000000000000000000a040" { print $2; exit }' "$work/go.g1")
accumulator_read=$(awk '$2 == "000000000000000000000000000a0430" { print $4; exit }' "$work/go.g1")
element_read=$(awk '$2 == "000000000000000000000000000a0431" { print $4; exit }' "$work/go.g1")

reject_graph() {
  input=$1
  name=$2
  if "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$input" "$work/$name.seme" >/dev/null 2>&1; then
    if "$work/lower" "$work/$name.seme" "$work/$name.wasm" "$work/$name.json" >/dev/null 2>&1; then
      echo "$name corruption was accepted" >&2
      exit 1
    fi
  fi
}

sed "s/fi 000000000000000000000000000a0401 rf $integer/fi 000000000000000000000000000a0401 rf $boolean/" \
  "$work/go.g1" > "$work/wrong-map-value.g1"
reject_graph "$work/wrong-map-value.g1" wrong-map-value

sed "s/fi 000000000000000000000000000a0410 rf $map_type/fi 000000000000000000000000000a0410 rf $integer/" \
  "$work/go.g1" > "$work/wrong-empty-type.g1"
reject_graph "$work/wrong-empty-type.g1" wrong-empty-type

awk -v from="$accumulator_read" -v replacement="$element_read" '
  $1 == "fi" && $2 == "000000000000000000000000000a0430" && $4 == from && !done { print $1, $2, "rf", replacement; done = 1; next }
  { print }
' "$work/go.g1" > "$work/wrong-update-map.g1"
reject_graph "$work/wrong-update-map.g1" wrong-update-map

awk -v replacement="$element_read" '
  $1 == "fi" && $2 == "000000000000000000000000000a0421" { seen++ }
  seen == 2 && $1 == "fi" && $2 == "000000000000000000000000000a0421" { print $1, $2, "rf", replacement; seen++; next }
  { print }
' "$work/go.g1" > "$work/wrong-final-key.g1"
reject_graph "$work/wrong-final-key.g1" wrong-final-key

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/tally.wasm" "$work/pulp/pure-function.wasm"
payload=0300000000000000feffffffffffffff03000000000000000700000000000000feffffffffffffff0300000000000000
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 \
  -request "10000000060000000300000000000000$payload" > "$work/pulp-repeated.log" 2>&1
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 \
  -request "1000000006000000feffffffffffffff$payload" > "$work/pulp-negative.log" 2>&1
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 \
  -request "10000000060000006300000000000000$payload" > "$work/pulp-missing.log" 2>&1
rg -q '"response":"0300000000000000"' "$work/pulp-repeated.log"
rg -q '"response":"0200000000000000"' "$work/pulp-negative.log"
rg -q '"response":"0000000000000000"' "$work/pulp-missing.log"

version=2
while [ "$version" -le 29 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo "Core Execution v30: Go and JavaScript runtime-keyed map folds agreed through Wasm and Pulp"
