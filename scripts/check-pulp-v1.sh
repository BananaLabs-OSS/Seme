#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_proof_commit=97b1c8c
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-pulp-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

if [ ! -f "$pulp_repo/go.mod" ] || [ ! -f "$pulp_repo/cmd/pulp-seme-proof/main.go" ]; then
    echo "Pulp proof deployment is unavailable at $pulp_repo" >&2
    exit 65
fi
if ! git -C "$pulp_repo" merge-base --is-ancestor "$pulp_proof_commit" HEAD; then
    echo "Pulp does not contain required proof commit $pulp_proof_commit" >&2
    exit 65
fi

(cd "$repo/targets/wasm/pulp-v1" && sha256sum -c quota-admit.wasm.sha256)
(cd "$pulp_repo" && go test ./cmd/pulp-seme-proof && go build -o "$work/pulp-seme-proof" ./cmd/pulp-seme-proof)

# The generated reactor must start through Pulp's real manifest loader, wazero
# runtime, capability registry, lifecycle calls, and step loop. GNU timeout
# supplies SIGINT and preserves Pulp's clean zero exit after the proof event.
timeout --preserve-status --signal=INT 1s "$work/pulp-seme-proof" \
    -manifest "$repo/targets/wasm/pulp-v1/pulp.cell.toml" \
    > "$work/allowed.log" 2>&1
rg -q 'cell ready.*cell=seme-quota' "$work/allowed.log"
rg -q '\[observability.log\] cell=seme-quota quota.accepted=true' "$work/allowed.log"
test "$(rg -c '^\[observability.log\]' "$work/allowed.log")" -eq 1
rg -q 'pulp exit clean' "$work/allowed.log"

# The identical artifact without the manifest grant receives Pulp's gated stub.
# The guest treats its nonzero denial status as fatal and initialization fails.
set +e
"$work/pulp-seme-proof" \
    -manifest "$repo/targets/wasm/pulp-v1/pulp.denied.cell.toml" \
    > "$work/denied.log" 2>&1
denied_status=$?
set -e
test "$denied_status" -eq 1
rg -q 'init failed.*cell=seme-quota-denied' "$work/denied.log"
rg -q 'wasm error: unreachable' "$work/denied.log"
if rg -q '^\[observability.log\]' "$work/denied.log"; then
    echo "denied Pulp cell performed the logging effect" >&2
    exit 1
fi

echo "Pulp target v1: generated Seme Wasm ran with granted effect and trapped when denied"
