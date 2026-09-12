#!/bin/sh
# Execute one source-free UPB12 authority through all three native projections,
# canonical Seme, standalone Wasm, and the pinned public Pulp host.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
authority=${1:?source-free UPB12 authority required}
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-upb12-runtime.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME

node "$repo/reference/js/upb12-runtime-corpus.mjs" "$work/vectors.json" "$work/requests.jsonl" "$work/expected.jsonl"
test "$(wc -l < "$work/requests.jsonl" | tr -d ' ')" -eq 4096
"$repo/scripts/project-upb12-authority.sh" "$authority" seme.upb12/service "$work/native"

(cd "$work/native/go" && go test -count=1 ./... -args -upb12-native-output "$work/go-native.jsonl")
node "$repo/reference/js/javascript-upb09-native.mjs" "$work/native/javascript" "$work/vectors.json" > "$work/javascript-native.jsonl"
env XDG_DATA_HOME="$work/lua-data" XDG_STATE_HOME="$work/lua-state" XDG_CACHE_HOME="$work/lua-cache" \
  nvim --clean -u NONE -n --headless -l "$repo/reference/lua/lua-upb09-native.lua" \
  "$work/native/lua" "$work/vectors.json" "$work/native/lua/seme-values.lua" > "$work/lua-native.jsonl"
for language in go javascript lua; do
  node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/expected.jsonl" "$work/$language-native.jsonl"
done

(cd "$repo/reference/go" &&
  go build -buildvcs=false -o "$work/entry-view" ./cmd/upb12-entry-view &&
  go build -buildvcs=false -o "$work/observe" ./cmd/canonical-observe &&
  go build -buildvcs=false -o "$work/codec" ./cmd/pure-application-codec &&
  GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -buildvcs=false -o "$work/vm.wasm" ./cmd/canonical-wasm-cell)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$authority/construction-v36.g1" "$work/project.seme"
"$work/entry-view" -authority "$authority" -graph "$work/project.seme" -out "$work/program.seme"
"$work/observe" "$work/program.seme" < "$work/requests.jsonl" > "$work/canonical.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/expected.jsonl" "$work/canonical.jsonl"
"$work/codec" encode "$work/program.seme" < "$work/requests.jsonl" > "$work/requests.hex"
node "$repo/reference/js/canonical-wasm-cell-runner.mjs" "$work/vm.wasm" "$work/program.seme" "$work/requests.hex" > "$work/wasm.hex"
"$work/codec" observe "$work/program.seme" < "$work/wasm.hex" > "$work/wasm.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/expected.jsonl" "$work/wasm.jsonl"

test -f "$pulp_repo/go.mod"; git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pinned" "$work/pulp"; git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pinned"
mkdir -p "$work/pinned/cmd/pulp-seme-canonical-vm-proof"
cp "$repo/targets/wasm/pulp-canonical-vm-v1/runner.go" "$work/pinned/cmd/pulp-seme-canonical-vm-proof/main.go"
(cd "$work/pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-canonical-vm-proof)
cp "$repo/targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/vm.wasm" "$work/pulp/canonical-vm.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -graph "$work/program.seme" -requests "$work/requests.hex" > "$work/pulp.raw.jsonl"
node "$repo/reference/js/pulp-canonical-vm-output.mjs" "$work/pulp.raw.jsonl" > "$work/pulp.hex"
"$work/codec" observe "$work/program.seme" < "$work/pulp.hex" > "$work/pulp.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/expected.jsonl" "$work/pulp.jsonl"

sed -n '1p' "$work/requests.hex" > "$work/one.hex"
if node "$repo/reference/js/canonical-wasm-cell-runner.mjs" "$work/vm.wasm" "$work/program.seme" "$work/one.hex" --deny > "$work/denied.out" 2> "$work/denied.err"; then
  echo 'UPB12 standalone Wasm accepted a denied capability' >&2; exit 1
fi
test ! -s "$work/denied.out"; rg -q canonical_vm.effect_denied "$work/denied.err"
printf 'UPB12 runtime: Go, JavaScript, Lua, canonical Seme, standalone Wasm, and pinned Pulp agree over 4,096 observations\n'
