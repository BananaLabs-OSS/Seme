#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

# The projector test independently performs native multi-file Go lifting,
# direct canonical-to-Go projection, gofmt/type checking, native execution,
# byte-identical re-lifting, and fail-closed projection rejection.
(cd "$repo/reference/go" && go test -count=1 -buildvcs=false ./goprojector)

# The cumulative gate supplies target parity for the same typed package-call
# semantics through deterministic Wasm and pinned Pulp, and rejects a call
# whose target is absent from canonical Program membership.
"$repo/scripts/check-core-cumulative-v30.sh"

echo "UAB-v1 Go UAB-01: lift, native parity, target parity, projection round trip, and rejection passed"
