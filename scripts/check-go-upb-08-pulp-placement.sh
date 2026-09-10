#!/bin/sh
# Authenticated placement evidence for Ordered Transport v1. The only accepted
# Pulp role is a pure semantic planner carried synchronously over opaque
# pulp_on_call bytes. This gate deliberately proves that this is not a framed
# or authoritative TransportPort realization.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
manifest="$repo/targets/wasm/pulp-function-composite-v1/pulp.cell.toml"
manifest_sha=e88c5c9fb7d4c839090dcbc546d838466d6a68b8d818ce3fa9e8da7efeea826c
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb08-pulp-placement.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME

test -f "$pulp_repo/go.mod"
git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
test "$(git -C "$pulp_repo" rev-parse "$pulp_commit^{commit}")" = "$pulp_commit"
mkdir "$work/pinned" "$work/cell"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pinned"

# Authenticate the exact capability-free provider declaration. Any manifest
# mutation, capability grant, consume edge, or extension selection fails.
test "$(sha256sum "$manifest" | cut -d ' ' -f 1)" = "$manifest_sha"
node - "$manifest" <<'NODE'
const fs = require("fs");
const text = fs.readFileSync(process.argv[2], "utf8");
function one(pattern, label) {
  const hits = [...text.matchAll(pattern)];
  if (hits.length !== 1) throw Error(`${label}: expected one declaration`);
  return hits[0][1];
}
if (one(/^provides\s*=\s*\[(.*)\]\s*$/gm, "provides").trim() !== '"seme.function-composite-v1"') throw Error("unexpected provider");
if (one(/^consumes\s*=\s*\[(.*)\]\s*$/gm, "consumes").trim() !== "") throw Error("planner consumes another provider");
if (one(/^capabilities\s*=\s*\[(.*)\]\s*$/gm, "capabilities").trim() !== "") throw Error("planner has capabilities");
NODE

# Authenticate the source feature being claimed, and reject accidental claims
# that the same revision contains the Ordered Transport contract/provider.
rg -q 'ExportedFunction\("pulp_on_call"\)' "$work/pinned/internal/host/cell.go"
rg -q 'for _, request := range options.requests' "$work/pinned/cmd/pulp-seme-function-proof/main.go"
rg -q 'cell.Call\(ctx, options.provider, request\)' "$work/pinned/cmd/pulp-seme-function-proof/main.go"
if rg -n 'seme\.transport\.(receive|send)\.v1|Ordered Transport|TransportPort|SEMEOT01' "$work/pinned"; then
  echo 'pinned Pulp unexpectedly claims an Ordered Transport provider' >&2
  exit 1
fi

# Exercise the local verifier's complete rejection matrix (StepEvent,
# HTTP/WebSocket, fs/sqlite, stdio, raw calls, and unpinned extensions).
(cd "$repo/reference/go" && go test -count=1 ./orderedtransportplacement)

# Exercise the pinned capability resolver itself: names resembling transport
# operations cannot be inferred from unrelated registered mechanisms.
cat > "$work/pinned/ext/upb08_transport_placement_test.go" <<'EOF'
package ext

import "testing"

func TestUPB08PinnedPulpHasNoAuthoritativeTransportProvider(t *testing.T) {
	registered := []Capability{
		{Name: "http", Provider: "pulp.http"},
		{Name: "websocket", Provider: "pulp.websocket"},
		{Name: "storage.fs", Provider: "github.com/BananaLabs-OSS/Pulp-ext-fs"},
		{Name: "storage.sqlite", Provider: "github.com/BananaLabs-OSS/Pulp-ext-sqlite"},
	}
	for _, operation := range []string{"seme.transport.receive.v1", "seme.transport.send.v1"} {
		if selected, err := SelectCapabilities(registered, map[string]string{operation: "seme.ordered-transport.v1"}); err == nil {
			t.Fatalf("resolver falsely mapped %q: %#v", operation, selected)
		}
	}
}
EOF
(cd "$work/pinned" && go test -count=1 ./ext -run '^TestUPB08PinnedPulpHasNoAuthoritativeTransportProvider$')

# Lower a small existing neutral composite program and run three distinct
# requests through the pinned runner in one process. Its JSON audit echoes the
# exact accepted request bytes; comparing the complete ordered sequence proves
# opaque byte preservation and input order for pulp_on_call only.
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" \
  "$repo/conformance/uab-v1/composite-option-result.g1" "$work/program.seme"
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/lower" ./cmd/composite-wasm-lower)
"$work/lower" "$work/program.seme" "$work/cell/pure-function.wasm" "$work/abi.json"
cp "$manifest" "$work/cell/pulp.cell.toml"
(cd "$work/pinned" && go build -buildvcs=false -o "$work/runner" ./cmd/pulp-seme-function-proof)
(cd "$work/cell" && "$work/runner" -manifest pulp.cell.toml -provider seme.function-composite-v1 \
  -request 00000000000000000000 \
  -request 01000a000000020000006f6b \
  -request 01010a00000003000000626164) > "$work/audit.jsonl"
node - "$work/audit.jsonl" <<'NODE'
const fs = require("fs");
const rows = fs.readFileSync(process.argv[2], "utf8").trim().split("\n").map(JSON.parse);
const expected = [
  ["00000000000000000000", "00"],
  ["01000a000000020000006f6b", "01"],
  ["01010a00000003000000626164", "01"],
];
if (rows.length !== expected.length) throw Error(`got ${rows.length} calls`);
for (let i = 0; i < expected.length; i++) {
  if (rows[i].request !== expected[i][0]) throw Error(`request ${i} changed or reordered`);
  if (rows[i].response !== expected[i][1]) throw Error(`response ${i} differs`);
}
NODE

echo 'Go UPB-08 pinned Pulp placement: pure synchronous opaque planner verified; authoritative TransportPort rejected'
