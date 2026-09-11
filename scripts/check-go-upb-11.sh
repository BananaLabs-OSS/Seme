#!/bin/sh
# Cumulative Go UPB-11 seven-class project reconciliation gate.
# This is the sole script permitted to claim go/UPB-11.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb11.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"
export GOCACHE XDG_CACHE_HOME

# Preserve and reproduce the complete claimed predecessor, including all
# seven evidence classes, 92 target requirements, Wasm, and Pulp authority.
"$repo/scripts/check-go-upb-10.sh"

# Reproduce every neutral contract added by the reconciliation layer.
"$repo/scripts/check-language-service-v1.sh"
"$repo/scripts/check-project-contract-v14.sh"

# Exercise the reusable transaction, identity-continuity, atomic bundle,
# source-free finalizer, and fail-closed command surfaces independently.
(cd "$repo/reference/go" && go test -p=1 -count=1 \
  ./patchinstance ./projectv14instance ./goprovider ./goupb11pipeline \
  ./cmd/go-upb09-build ./cmd/go-upb11-reconcile ./cmd/go-upb11-finalize)

# Prove the bounded mechanism independently, then use it on the complete real
# cumulative fixture and regenerate both target deployments and Project-v14.
"$repo/scripts/check-go-upb-11-reconciliation.sh"
"$repo/scripts/check-go-upb-11-project.sh"

printf 'Go UPB-11 seven-class transactional project reconciliation authority gate passes\n'
