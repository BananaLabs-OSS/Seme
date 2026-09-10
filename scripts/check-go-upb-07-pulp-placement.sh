#!/bin/sh
# Negative placement evidence against the exact Pulp revision used by the
# cumulative target gate. This proves non-placement; it does not emulate a
# DurablePort provider.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
manifest="$repo/targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml"
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb07-pulp-placement.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; export GOCACHE

test -f "$pulp_repo/go.mod"
git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pulp"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp"
test "$(git -C "$pulp_repo" rev-parse "$pulp_commit^{commit}")" = "$pulp_commit"

# The planner target grants exactly logging. A storage grant added beside it
# must make this check fail rather than silently changing placement evidence.
node - "$manifest" <<'NODE'
const fs=require("fs");
const text=fs.readFileSync(process.argv[2],"utf8");
const matches=[...text.matchAll(/^capabilities\s*=\s*\[(.*)\]\s*$/gm)];
if(matches.length!==1)throw Error("planner manifest must have one capabilities assignment");
const names=[...matches[0][1].matchAll(/"([^"]+)"/g)].map(x=>x[1]);
if(JSON.stringify(names)!=='["observability.log"]')throw Error(`planner capabilities are ${JSON.stringify(names)}`);
NODE

# Pulp's pinned module names its available storage families, while the pinned
# source contains neither Durable-v1 operation identity. Filesystem read/write
# and SQLite query/exec therefore are not silently treated as opaque-token
# Load/CompareExchange.
rg -q 'Pulp-ext-fs' "$work/pulp/go.mod"
rg -q 'Pulp-ext-sqlite' "$work/pulp/go.mod"
if rg -n 'seme\.storage\.(load|compare_exchange)\.v1|DurablePort' "$work/pulp"; then
  echo 'pinned Pulp unexpectedly reports a Durable-v1 mapping' >&2
  exit 1
fi

# Exercise the pinned resolver itself. Even when its two advertised storage
# capability names are registered, exact Durable-v1 names must fail closed.
cat > "$work/pulp/ext/upb07_placement_test.go" <<'EOF'
package ext

import "testing"

func TestUPB07PinnedStorageDoesNotResolveDurableOperations(t *testing.T) {
	registered := []Capability{
		{Name: "observability.log", Provider: "seme.pulp.log-v1"},
		{Name: "storage.fs", Provider: "github.com/BananaLabs-OSS/Pulp-ext-fs"},
		{Name: "storage.sqlite", Provider: "github.com/BananaLabs-OSS/Pulp-ext-sqlite"},
	}
	for _, operation := range []string{"seme.storage.load.v1", "seme.storage.compare_exchange.v1"} {
		selected, err := SelectCapabilities(registered, map[string]string{operation: "seme.durable-state.v1"})
		if err == nil {
			t.Fatalf("resolver falsely mapped %q: %#v", operation, selected)
		}
	}
}
EOF
(cd "$work/pulp" && go test -count=1 ./ext -run '^TestUPB07PinnedStorageDoesNotResolveDurableOperations$')

echo 'Go UPB-07 pinned Pulp placement: planner grants logging only and exact Durable-v1 Load/CAS resolution fails closed'
