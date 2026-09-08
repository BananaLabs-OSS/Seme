#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v29.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
 go run -buildvcs=false ./cmd/execution-module-v29 > "$work/module.g1" && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
cmp "$repo/modules/execution/v29/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v29" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v29/module.seme" "$work/module.seme"
(cd "$repo/fixtures/go-execution-v29" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test && node mutable-closure-native-runner.mjs)

"$work/session" --module "$repo/modules/execution/v29/module.g1" \
  --project "$repo/fixtures/go-execution-v29" --package example.test/mutable-closure \
  --entry Run --revision 1 --out "$work/go.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/go.g1" "$work/go.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/go.g1" "$work/projected.mjs"
node --check "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v29/module.g1" --package example.test/mutable-closure \
  --entry Run --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/go.seme" "$work/js.seme"

"$work/lower" "$work/go.seme" "$work/run.wasm" "$work/abi.json"
"$work/lower" "$work/go.seme" "$work/run-second.wasm" "$work/abi-second.json"
cmp "$work/run.wasm" "$work/run-second.wasm"
cmp "$work/abi.json" "$work/abi-second.json"
rg -q '"request_size": 24' "$work/abi.json"
rg -q '"response_size": 8' "$work/abi.json"
node "$repo/reference/js/mutable-closure-runner.mjs" "$work/run.wasm"

capture=$(awk '$3 == "0000000000000000000000000000a030" { print $2; exit }' "$work/go.g1")
parameter=$(awk '$3 == "00000000000000000000000000009012" { print $2; exit }' "$work/go.g1")
stateful_call=$(awk '$3 == "0000000000000000000000000000a035" { print $2; exit }' "$work/go.g1")
parameter_read=$(awk '$3 == "00000000000000000000000000009013" { print $2; exit }' "$work/go.g1")

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

awk -v capture="$capture" '
  $1 == "en" && $3 == "0000000000000000000000000000a034" { inside = 1 }
  inside && $1 == "fi" && $2 == "000000000000000000000000000a0342" { print $1, $2, "li 0"; drop = 1; next }
  inside && drop && $1 == "rf" && $2 == capture { drop = 0; next }
  { print }
' "$work/go.g1" > "$work/missing-capture.g1"
reject_graph "$work/missing-capture.g1" missing-capture

sed "s/fi 000000000000000000000000000a0320 rf $capture/fi 000000000000000000000000000a0320 rf $parameter/" \
  "$work/go.g1" > "$work/wrong-update-target.g1"
reject_graph "$work/wrong-update-target.g1" wrong-update-target

awk -v replacement="$stateful_call" '
  $1 == "fi" && $2 == "00000000000000000000000000009e31" && !done { print $1, $2, "rf", replacement; done = 1; next }
  { print }
' "$work/go.g1" > "$work/wrong-commit-link.g1"
reject_graph "$work/wrong-commit-link.g1" wrong-commit-link

awk -v replacement="$parameter_read" '
  $1 == "fi" && $2 == "000000000000000000000000000a0350" && !done { print $1, $2, "rf", replacement; done = 1; next }
  { print }
' "$work/go.g1" > "$work/wrong-call-link.g1"
reject_graph "$work/wrong-call-link.g1" wrong-call-link

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/run.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 \
  -request 0a0000000000000005000000000000000700000000000000 > "$work/pulp-positive.log" 2>&1
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 \
  -request f6fffffffffffffffBffffffffffffff0700000000000000 > "$work/pulp-negative.log" 2>&1
rg -q '"response":"1600000000000000"' "$work/pulp-positive.log"
rg -q '"response":"f8ffffffffffffff"' "$work/pulp-negative.log"

version=2
while [ "$version" -le 28 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo "Core Execution v29: Go and JavaScript mutable closures committed explicit environments through Wasm and Pulp"
