#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v28.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
 go run -buildvcs=false ./cmd/execution-module-v28 > "$work/module.g1" && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
cmp "$repo/modules/execution/v28/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v28" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v28/module.seme" "$work/module.seme"
(cd "$repo/fixtures/go-execution-v28" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test && node immutable-closure-native-runner.mjs)

lift() {
  entry=$1
  output=$2
  "$work/session" --module "$repo/modules/execution/v28/module.g1" \
    --project "$repo/fixtures/go-execution-v28" --package example.test/immutable-closure \
    --entry "$entry" --revision 1 --out "$work/$output.g1"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/$output.g1" "$work/$output.seme"
}

lift Run run
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/run.g1" "$work/projected.mjs"
node --check "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v28/module.g1" --package example.test/immutable-closure \
  --entry Run --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/run.seme" "$work/js.seme"

"$work/lower" "$work/run.seme" "$work/run.wasm" "$work/run-abi.json"
"$work/lower" "$work/run.seme" "$work/run-second.wasm" "$work/run-second-abi.json"
cmp "$work/run.wasm" "$work/run-second.wasm"
cmp "$work/run-abi.json" "$work/run-second-abi.json"
rg -q '"request_size": 16' "$work/run-abi.json"
rg -q '"response_size": 8' "$work/run-abi.json"
node "$repo/reference/js/immutable-closure-runner.mjs" "$work/run.wasm"

lift MakeAdder make
lift Apply apply
"$work/lower" "$work/make.seme" "$work/make.wasm" "$work/make-abi.json"
"$work/lower" "$work/apply.seme" "$work/apply.wasm" "$work/apply-abi.json"
rg -q '"request_size": 8' "$work/make-abi.json"
rg -q '"response_size": 16' "$work/make-abi.json"
rg -q '"request_size": 24' "$work/apply-abi.json"
rg -q '"response_size": 8' "$work/apply-abi.json"
node "$repo/reference/js/closure-abi-runner.mjs" "$work/make.wasm" "$work/apply.wasm"

integer=$(awk '$3 == "00000000000000000000000000009010" { print $2; exit }' "$work/run.g1")
boolean=$(awk '$3 == "00000000000000000000000000009020" { print $2; exit }' "$work/run.g1")
capture=$(awk '$3 == "0000000000000000000000000000a021" { print $2; exit }' "$work/run.g1")

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

sed "s/fi 000000000000000000000000000a0211 rf $integer/fi 000000000000000000000000000a0211 rf $boolean/" \
  "$work/run.g1" > "$work/wrong-capture-type.g1"
reject_graph "$work/wrong-capture-type.g1" wrong-capture-type

sed "s/fi 000000000000000000000000000a0201 rf $integer/fi 000000000000000000000000000a0201 rf $boolean/" \
  "$work/run.g1" > "$work/wrong-function-result.g1"
reject_graph "$work/wrong-function-result.g1" wrong-function-result

awk -v capture="$capture" '
  $1 == "en" && $3 == "0000000000000000000000000000a023" { inside = 1 }
  inside && $1 == "fi" && $2 == "000000000000000000000000000a0232" { print $1, $2, "li 0"; drop = 1; next }
  inside && drop && $1 == "rf" && $2 == capture { drop = 0; next }
  { print }
' "$work/run.g1" > "$work/missing-capture.g1"
reject_graph "$work/missing-capture.g1" missing-capture

awk '
  $1 == "en" && $3 == "0000000000000000000000000000a024" { inside = 1 }
  inside && $1 == "fi" && $2 == "000000000000000000000000000a0241" { print $1, $2, "li 0"; drop = 1; next }
  inside && drop && $1 == "rf" { drop = 0; next }
  { print }
' "$work/run.g1" > "$work/wrong-indirect-arity.g1"
reject_graph "$work/wrong-indirect-arity.g1" wrong-indirect-arity

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/run.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 \
  -request 07000000000000000500000000000000 > "$work/pulp-positive.log" 2>&1
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 \
  -request f9ffffffffffffff0500000000000000 > "$work/pulp-negative.log" 2>&1
rg -q '"response":"0c00000000000000"' "$work/pulp-positive.log"
rg -q '"response":"feffffffffffffff"' "$work/pulp-negative.log"

cp "$work/make.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 \
  -request f9ffffffffffffff > "$work/pulp-make.log" 2>&1
rg -q '"response":"0100000000000000f9ffffffffffffff"' "$work/pulp-make.log"
cp "$work/apply.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 \
  -request 0100000000000000f9ffffffffffffff0c00000000000000 > "$work/pulp-apply.log" 2>&1
rg -q '"response":"0500000000000000"' "$work/pulp-apply.log"

version=2
while [ "$version" -le 27 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo "Core Execution v28: Go and JavaScript immutable closures retained explicit environments through Wasm and Pulp"
