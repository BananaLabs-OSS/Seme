#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-uab-10.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
provider="$repo/modules/provider/v1/module.g1"
go_provider="$work/go-provider"

(cd "$repo/reference/go" && go build -o "$go_provider" ./cmd/go-provider)
cp -R "$repo/fixtures/go-uab-10" "$work/project"
(cd "$work/project" && go test ./...)

"$go_provider" ingest --project "$work/project" --module "$provider" --out "$work/before"
(cd "$work/project" && gofmt -w identity.go && go test ./...)
"$go_provider" ingest --project "$work/project" --module "$provider" --prior "$work/before/manifest.json" --out "$work/formatted"
(cd "$repo/reference/go" && go run ./cmd/provider-identity-check "$work/before/manifest.json" "$work/formatted/manifest.json")

# The existing contract gate performs the canonical Patch rename, atomic
# minimal projection, native validation, deterministic re-import, and stale
# revision/unstamped-candidate rejection.
"$repo/scripts/check-provider-v1.sh"

# Two indistinguishable declarations whose native keys both change cannot be
# reconciled honestly. The provider must reject rather than guess an identity,
# and the diagnostic must identify the declaration and source position.
cp -R "$repo/fixtures/go-uab-10-ambiguous" "$work/ambiguous"
"$go_provider" ingest --project "$work/ambiguous" --module "$provider" --out "$work/ambiguous-before"
(cd "$repo/reference/go" && go run ./cmd/provider-identity-check forge-ambiguous "$work/ambiguous-before/manifest.json" "$work/ambiguous-forged.json")
sed -i 's/Alpha/Gamma/g; s/Beta/Delta/g' "$work/ambiguous/functions.go"
if "$go_provider" ingest --project "$work/ambiguous" --module "$provider" --prior "$work/ambiguous-forged.json" --out "$work/ambiguous-after" 2>"$work/ambiguous.log"; then
	echo "ambiguous identity evidence was accepted" >&2
	exit 1
fi
grep -Eq 'provider.identity_ambiguous:[A-Za-z]+:functions.go:[1-9][0-9]*:[1-9][0-9]*' "$work/ambiguous.log"

# Deterministic Wasm and pinned Pulp remain execution targets for the same
# typed function/call family whose stable identities are proven above.
"$repo/scripts/check-execution-v16.sh"

echo "Go UAB-10 complete acceptance: all five evidence classes pass"
