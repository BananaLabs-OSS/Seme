#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-application-demo.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

if ! git -C "$pulp_repo" merge-base --is-ancestor 161c9dc HEAD; then
    echo "Pulp does not contain the required Seme application driver" >&2
    exit 65
fi
(cd "$repo/targets/wasm/pulp-v1" && sha256sum -c quota-admit.wasm.sha256)
(cd "$pulp_repo" && go build -o "$work/pulp-seme-proof" ./cmd/pulp-seme-proof)

echo
echo "Seme Application Proof v1"
echo "Go origin -> canonical Seme plan -> Seme Wasm -> Pulp provider calls"
echo
"$work/pulp-seme-proof" \
    -manifest "$repo/targets/wasm/pulp-v1/pulp.cell.toml" \
    -request 40,2,50 \
    -request 40,20,50 \
    -request 9223372036854775807,1,0

echo
echo "Capability denial (the identical Wasm must fail before logging):"
set +e
"$work/pulp-seme-proof" \
    -manifest "$repo/targets/wasm/pulp-v1/pulp.denied.cell.toml" \
    -request 40,2,50 \
    > "$work/denied.log" 2>&1
status=$?
set -e
test "$status" -eq 1
if rg -q '^\[observability.log\]' "$work/denied.log"; then
    echo "denied cell performed the forbidden effect" >&2
    exit 1
fi
rg 'pulp_on_call trap: wasm error: unreachable' "$work/denied.log"
echo "Denied as required."
