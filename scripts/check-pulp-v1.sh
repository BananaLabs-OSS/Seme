#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_proof_commit=c303c74
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

# The generated reactor must process multiple structured requests through
# Pulp's real manifest loader, capability registry, cell lifecycle, allocator,
# and pulp_on_call provider ABI without being reloaded between requests.
"$work/pulp-seme-proof" \
    -manifest "$repo/targets/wasm/pulp-v1/pulp.cell.toml" \
    -request 40,2,50 \
    -request 40,20,50 \
    -request 9223372036854775807,1,0 \
    -request 40,2,50, \
    > "$work/allowed.log" 2>&1
rg -q '\[observability.log\] cell=seme-quota quota.accepted=true' "$work/allowed.log"
rg -q '\[observability.log\] cell=seme-quota quota.accepted=false' "$work/allowed.log"
test "$(rg -c '^\[observability.log\]' "$work/allowed.log")" -eq 3
test "$(rg -o '"subject":"tenant-a"' "$work/allowed.log" | wc -l)" -eq 6
test "$(rg -o '"evidence":"AQID"' "$work/allowed.log" | wc -l)" -eq 7
rg -q '"current":40,"delta":2,"limit":50.*"accepted":true' "$work/allowed.log"
rg -q '"current":40,"delta":20,"limit":50.*"accepted":false' "$work/allowed.log"
rg -q '"current":9223372036854775807,"delta":1,"limit":0.*"accepted":true' "$work/allowed.log"
rg -q '"subject":"".*"error":"subject required"' "$work/allowed.log"
rg -q 'shutdown complete.*cell=seme-quota' "$work/allowed.log"

# The identical artifact without the manifest grant receives Pulp's gated stub.
# The guest treats its nonzero denial status as fatal and the call traps.
set +e
"$work/pulp-seme-proof" \
    -manifest "$repo/targets/wasm/pulp-v1/pulp.denied.cell.toml" \
    -request 40,2,50 \
    > "$work/denied.log" 2>&1
denied_status=$?
set -e
test "$denied_status" -eq 1
rg -q 'pulp_on_call trap: wasm error: unreachable' "$work/denied.log"
if rg -q '^\[observability.log\]' "$work/denied.log"; then
    echo "denied Pulp cell performed the logging effect" >&2
    exit 1
fi

echo "Pulp target v1: ResultOk/ResultError requests ran with granted effect and trapped when denied"
