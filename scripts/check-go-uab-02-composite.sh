#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-uab02-composite.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME
GOCACHE="$work/go-build"; export GOCACHE

(cd "$repo/reference/go" && go test -count=1 -buildvcs=false ./goprovider ./goprojector ./wasmtarget && \
  go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
  go build -buildvcs=false -o "$work/projector" ./cmd/go-projector && \
  go build -buildvcs=false -o "$work/lower" ./cmd/composite-wasm-lower && \
  go build -buildvcs=false -o "$work/eval" ./cmd/canonical-eval)

(cd "$repo/fixtures/go-uab-02-composite" && go test -count=1 -buildvcs=false ./... && \
  go run -buildvcs=false ./cmd/native-observations > "$work/native.json")
expected='{"none":false,"ok":true,"wrong_bytes":false,"error":true,"wrong_error":false}'
test "$(tr -d '\n' < "$work/native.json")" = "$expected"

"$work/session" --module "$repo/modules/execution/v32/module.g1" \
  --project "$repo/fixtures/go-uab-02-composite" --package example.test/go-uab-02-composite \
  --entry Admit --revision 1 --out "$work/go.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/go.g1" "$work/go.seme"
"$work/eval" "$work/go.seme" "$repo/fixtures/go-uab-02-composite/canonical-vectors.json" > "$work/canonical.json"
rg -q '"malformed":3' "$work/canonical.json"
"$work/projector" -package composite "$work/go.g1" "$work/projected.go"

mkdir -p "$work/projected/cmd/native-observations"
cp "$repo/fixtures/go-uab-02-composite/go.mod" "$work/projected/go.mod"
cp "$work/projected.go" "$work/projected/function.go"
cp "$repo/fixtures/go-uab-02-composite/cmd/native-observations/main.go" "$work/projected/cmd/native-observations/main.go"
(cd "$work/projected" && go test -count=1 -buildvcs=false ./... && \
  go run -buildvcs=false ./cmd/native-observations > "$work/projected.json")
cmp "$work/native.json" "$work/projected.json"

"$work/session" --module "$repo/modules/execution/v32/module.g1" \
  --project "$work/projected" --package example.test/go-uab-02-composite \
  --entry Admit --revision 1 --out "$work/relifted.g1"
cmp "$work/go.g1" "$work/relifted.g1"

"$work/lower" "$work/go.seme" "$work/program.wasm" "$work/abi.json"
node "$repo/reference/js/composite-function-runner.mjs" "$work/program.wasm" --json > "$work/wasm.json"
cmp "$work/native.json" "$work/wasm.json"

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-composite-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/program.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-composite-v1 \
  -request 00000000000000000000 \
  -request 01000a000000020000006f6b \
  -request 01000a000000020000006e6f \
  -request 01010a00000003000000626164 \
  -request 01010a000000020000006e6f > "$work/pulp.log" 2>&1
rg -q '"request":"00000000000000000000","response":"00"' "$work/pulp.log"
rg -q '"request":"01000a000000020000006f6b","response":"01"' "$work/pulp.log"
rg -q '"request":"01000a000000020000006e6f","response":"00"' "$work/pulp.log"
rg -q '"request":"01010a00000003000000626164","response":"01"' "$work/pulp.log"
rg -q '"request":"01010a000000020000006e6f","response":"00"' "$work/pulp.log"

for malformed in \
  02000000000000000000 \
  01020a00000000000000 \
  00010000000000000000 \
  0000000000000000000000 \
  0100090000000100000000 \
  01000a00000002000000 \
  01000b0000000100000000 \
  01010a00000002000000c328
do
  if "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-composite-v1 \
    -request "$malformed" > "$work/malformed.log" 2>&1; then
    echo "Pulp accepted malformed composite request $malformed" >&2
    exit 1
  fi
done

echo "Go UAB-02 composite: identical native/projected/canonical/Wasm/Pulp observations and malformed-vector rejection passed"
