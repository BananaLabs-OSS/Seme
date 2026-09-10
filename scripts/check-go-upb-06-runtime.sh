#!/bin/sh
# Pre-authority runtime evidence for the cumulative UPB-06 entry. The final
# UPB-06 gate must additionally authenticate Resource and Project v9 artifacts.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb06-runtime.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
proxy="$repo/fixtures/go-upb03-offline-proxy"
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001

"$repo/scripts/materialize-go-upb06-fixture.sh" "$work/source"
(cd "$work/source" && GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off go test ./service -run '^TestNativeResourceCorpus$' -args -native-resource-corpus-dir "$work/corpus")
(cd "$repo/reference/go" &&
  go build -buildvcs=false -o "$work/build" ./cmd/go-upb05-build &&
  go build -buildvcs=false -o "$work/observe" ./cmd/canonical-observe &&
  go build -buildvcs=false -o "$work/codec" ./cmd/pure-application-codec &&
  GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -buildvcs=false -o "$work/vm.wasm" ./cmd/canonical-wasm-cell)

"$work/build" -project "$work/source" -proxy "$proxy" -module example.test/go-uab-11 -package example.test/go-uab-11/service -entry ApplyConfiguredResource -revision 1 -out "$work/bundle" \
  -dependency example.test/seme/checksum -version v1.2.3 -local-from example.test/go-uab-11/application -local-to example.test/go-uab-11/model -selection "$repo/fixtures/go-upb05-configuration-overlay/configuration-selection.json" \
  -execution-g1 "$repo/modules/execution/v36/module.g1" -execution-contract "$repo/modules/execution/v36/module.seme" -foundation-contract "$repo/modules/foundation/v1/module.seme" \
  -package-v4 "$repo/modules/package/v4/module.seme" -project-v8 "$repo/modules/project/v8/module.seme" -dependency-v1 "$repo/modules/dependency/v1/module.seme" -configuration-v3 "$repo/modules/configuration/v3/module.seme" \
  -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0"

"$work/observe" "$work/bundle/execution-v36.seme" < "$work/corpus/requests.jsonl" > "$work/canonical.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/corpus/expected.jsonl" "$work/canonical.jsonl"
"$work/codec" encode "$work/bundle/execution-v36.seme" < "$work/corpus/requests.jsonl" > "$work/requests.hex"
node "$repo/reference/js/canonical-wasm-cell-runner.mjs" "$work/vm.wasm" "$work/bundle/execution-v36.seme" "$work/requests.hex" > "$work/wasm.hex"
"$work/codec" observe "$work/bundle/execution-v36.seme" < "$work/wasm.hex" > "$work/wasm.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/corpus/expected.jsonl" "$work/wasm.jsonl"

test -f "$pulp_repo/go.mod"
git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pinned" "$work/pulp"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pinned"
mkdir -p "$work/pinned/cmd/pulp-seme-canonical-vm-proof"
cp "$repo/targets/wasm/pulp-canonical-vm-v1/runner.go" "$work/pinned/cmd/pulp-seme-canonical-vm-proof/main.go"
(cd "$work/pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-canonical-vm-proof)
cp "$repo/targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/vm.wasm" "$work/pulp/canonical-vm.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -graph "$work/bundle/execution-v36.seme" -requests "$work/requests.hex" > "$work/pulp.raw.jsonl"
node "$repo/reference/js/pulp-canonical-vm-output.mjs" "$work/pulp.raw.jsonl" > "$work/pulp.hex"
"$work/codec" observe "$work/bundle/execution-v36.seme" < "$work/pulp.hex" > "$work/pulp.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/corpus/expected.jsonl" "$work/pulp.jsonl"
printf 'Go UPB-06 pre-authority runtime parity passes over 2,066 cases\n'
