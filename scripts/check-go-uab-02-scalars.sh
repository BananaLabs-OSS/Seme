#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}; pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-uab02-scalars.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; GOCACHE="$work/go-build"; export XDG_CACHE_HOME GOCACHE
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && go build -buildvcs=false -o "$work/projector" ./cmd/go-projector && go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower && go build -buildvcs=false -o "$work/eval" ./cmd/canonical-eval)
(cd "$repo/fixtures/go-uab-02-scalars" && go run -buildvcs=false ./cmd/native-observations > "$work/native.json")
mkdir "$work/pulp-pinned"; git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)

prove_entry() {
  entry=$1
  "$work/session" --module "$repo/modules/execution/v32/module.g1" --project "$repo/fixtures/go-uab-02-scalars" --package example.test/go-uab-02-scalars --entry "$entry" --revision 1 --out "$work/$entry.g1"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/$entry.g1" "$work/$entry.seme"
	case "$entry" in ObserveI64) vectors=i64;; ObserveBoolean) vectors=bool;; ObserveText) vectors=text;; esac
	"$work/eval" "$work/$entry.seme" "$repo/fixtures/go-uab-02-scalars/$vectors-canonical.json" > "$work/$entry-canonical.json"
  "$work/projector" -package scalars "$work/$entry.g1" "$work/$entry.go"
  mkdir -p "$work/$entry-project/cmd/native-observations"
  cp "$repo/fixtures/go-uab-02-scalars/go.mod" "$work/$entry-project/go.mod"; cp "$work/$entry.go" "$work/$entry-project/program.go"; cp "$repo/fixtures/go-uab-02-scalars/cmd/native-observations/main.go" "$work/$entry-project/cmd/native-observations/main.go"
  (cd "$work/$entry-project" && go run -buildvcs=false ./cmd/native-observations > "$work/$entry-native.json")
  cmp "$work/native.json" "$work/$entry-native.json"
  "$work/session" --module "$repo/modules/execution/v32/module.g1" --project "$work/$entry-project" --package example.test/go-uab-02-scalars --entry "$entry" --revision 1 --out "$work/$entry-relift.g1"
  cmp "$work/$entry.g1" "$work/$entry-relift.g1"
  "$work/lower" "$work/$entry.seme" "$work/$entry.wasm" "$work/$entry-abi.json"
  mkdir "$work/$entry-pulp"; cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/$entry-pulp/pulp.cell.toml"; cp "$work/$entry.wasm" "$work/$entry-pulp/pure-function.wasm"
}
prove_entry ObserveI64
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/ObserveI64.wasm" 0000000000000000 0100000000000000
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/ObserveI64.wasm" ffffffffffffff7f 0000000000000080
"$work/pulp-runner" -manifest "$work/ObserveI64-pulp/pulp.cell.toml" -provider seme.function-v2 -request 0000000000000000 -request ffffffffffffff7f > "$work/i64.log" 2>&1
rg -q '"response":"0100000000000000"' "$work/i64.log"; rg -q '"response":"0000000000000080"' "$work/i64.log"
prove_entry ObserveBoolean
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/ObserveBoolean.wasm" 00 00; node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/ObserveBoolean.wasm" 01 01
"$work/pulp-runner" -manifest "$work/ObserveBoolean-pulp/pulp.cell.toml" -provider seme.function-v2 -request 00 -request 01 > "$work/bool.log" 2>&1
rg -q '"response":"00"' "$work/bool.log"; rg -q '"response":"01"' "$work/bool.log"
prove_entry ObserveText
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/ObserveText.wasm" 0800000000000000 21
node "$repo/reference/js/pure-function-v2-runner.mjs" "$work/ObserveText.wasm" 080000000a000000e4b896e7958cf09f9a80 e4b896e7958cf09f9a8021
"$work/pulp-runner" -manifest "$work/ObserveText-pulp/pulp.cell.toml" -provider seme.function-v2 -request 0800000000000000 -request 080000000a000000e4b896e7958cf09f9a80 > "$work/text.log" 2>&1
rg -q '"response":"21"' "$work/text.log"; rg -q '"response":"e4b896e7958cf09f9a8021"' "$work/text.log"
echo "Go UAB-02 scalars: shared i64/Boolean/text vectors passed original/projected Go, canonical evaluation, byte-identical re-lift, standalone Wasm, and pinned Pulp"
