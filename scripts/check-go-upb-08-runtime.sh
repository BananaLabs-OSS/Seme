#!/bin/sh
# Pre-authority runtime evidence for cumulative UPB-08 ordered transport.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb08-runtime.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
proxy="$repo/fixtures/go-upb03-offline-proxy"
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001

"$repo/scripts/materialize-go-upb08-fixture.sh" "$work/project"
(cd "$work/project" && GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./streamservice -run '^TestNativeTransportRuntimeCorpus$' -args -native-transport-corpus-dir "$work/corpus-a")
(cd "$work/project" && GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./streamservice -run '^TestNativeTransportRuntimeCorpus$' -args -native-transport-corpus-dir "$work/corpus-b")
cmp "$work/corpus-a/requests.jsonl" "$work/corpus-b/requests.jsonl"
cmp "$work/corpus-a/expected.jsonl" "$work/corpus-b/expected.jsonl"
cmp "$work/corpus-a/COMPLETE" "$work/corpus-b/COMPLETE"
(cd "$repo/reference/go" &&
  go build -buildvcs=false -o "$work/lift" ./cmd/go-upb08-lift &&
  go build -buildvcs=false -o "$work/observe" ./cmd/canonical-observe &&
  go build -buildvcs=false -o "$work/codec" ./cmd/pure-application-codec &&
  GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -buildvcs=false -o "$work/vm.wasm" ./cmd/canonical-wasm-cell)
"$work/lift" -project "$work/project" -execution-g1 "$repo/modules/execution/v36/module.g1" -out "$work/program.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/program.g1" "$work/program.seme"
"$work/observe" "$work/program.seme" < "$work/corpus-a/requests.jsonl" > "$work/canonical.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/corpus-a/expected.jsonl" "$work/canonical.jsonl"
"$work/codec" encode "$work/program.seme" < "$work/corpus-a/requests.jsonl" > "$work/requests.hex"
node "$repo/reference/js/canonical-wasm-cell-runner.mjs" "$work/vm.wasm" "$work/program.seme" "$work/requests.hex" > "$work/wasm.hex"
"$work/codec" observe "$work/program.seme" < "$work/wasm.hex" > "$work/wasm.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/corpus-a/expected.jsonl" "$work/wasm.jsonl"

test -f "$pulp_repo/go.mod"
git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pinned" "$work/pulp"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pinned"
mkdir -p "$work/pinned/cmd/pulp-seme-canonical-vm-proof"
cp "$repo/targets/wasm/pulp-canonical-vm-v1/runner.go" "$work/pinned/cmd/pulp-seme-canonical-vm-proof/main.go"
(cd "$work/pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-canonical-vm-proof)
cp "$repo/targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/vm.wasm" "$work/pulp/canonical-vm.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -graph "$work/program.seme" -requests "$work/requests.hex" > "$work/pulp.raw.jsonl"
node "$repo/reference/js/pulp-canonical-vm-output.mjs" "$work/pulp.raw.jsonl" > "$work/pulp.hex"
"$work/codec" observe "$work/program.seme" < "$work/pulp.hex" > "$work/pulp.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/corpus-a/expected.jsonl" "$work/pulp.jsonl"
printf 'Go UPB-08 native, canonical, standalone Wasm, and Pulp parity passes over 4,096 ordered transport observations\n'
