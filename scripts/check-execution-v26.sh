#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v26.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
 go run -buildvcs=false ./cmd/execution-module-v26 > "$work/module.g1" && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
cmp "$repo/modules/execution/v26/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v26" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v26/module.seme" "$work/module.seme"
(cd "$repo/fixtures/go-execution-v26" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test && node javascript-method-transition-native-runner.mjs)

"$work/session" --module "$repo/modules/execution/v26/module.g1" \
  --project "$repo/fixtures/go-execution-v26" --package example.com/seme-method-transition-proof \
  --entry Step --revision 1 --out "$work/go.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/go.g1" "$work/go.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/go.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v26/module.g1" --package example.com/seme-method-transition-proof \
  --entry Step --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/go.seme" "$work/js.seme"

"$work/lower" "$work/go.seme" "$work/transition.wasm" "$work/abi.json"
rg -q '"request_size": 16' "$work/abi.json"
rg -q '"response_size": 16' "$work/abi.json"
rg -q 'state-transition:record:i64,i64' "$work/abi.json"
node "$repo/reference/js/state-transition-runner.mjs" "$work/transition.wasm"

method=$(awk '$3 == "0000000000000000000000000000a002" { print $2 }' "$work/go.g1")
record=$(awk '$3 == "00000000000000000000000000009030" { print $2 }' "$work/go.g1")
integer=$(awk '$3 == "00000000000000000000000000009010" { print $2 }' "$work/go.g1")
sed "s/fi 000000000000000000000000000a0031 rf $method/fi 000000000000000000000000000a0031 rf $record/" "$work/go.g1" > "$work/wrong-method.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/wrong-method.g1" "$work/wrong-method.seme"
if "$work/lower" "$work/wrong-method.seme" "$work/wrong-method.wasm" "$work/wrong-method.json" >/dev/null 2>&1; then
  echo "non-method call target was accepted" >&2
  exit 1
fi
sed "s/fi 000000000000000000000000000a0001 rf $record/fi 000000000000000000000000000a0001 rf $integer/" "$work/go.g1" > "$work/wrong-receiver.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/wrong-receiver.g1" "$work/wrong-receiver.seme"
if "$work/lower" "$work/wrong-receiver.seme" "$work/wrong-receiver.wasm" "$work/wrong-receiver.json" >/dev/null 2>&1; then
  echo "receiver type mismatch was accepted" >&2
  exit 1
fi

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/transition.wasm" "$work/pulp/pure-function.wasm"
request=f9ffffffffffffff0c00000000000000
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 -request "$request" > "$work/pulp.log" 2>&1
rg -q '"response":"05000000000000000500000000000000"' "$work/pulp.log"

version=2
while [ "$version" -le 25 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo "Core Execution v26: native Go and JavaScript methods shared one explicit state transition through Wasm and Pulp"
