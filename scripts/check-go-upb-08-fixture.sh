#!/bin/sh
# Pre-claim native fixture and deterministic corpus evidence for Go UPB-08.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb08-fixture.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; export GOCACHE
proxy="$repo/fixtures/go-upb03-offline-proxy"
"$repo/scripts/materialize-go-upb08-fixture.sh" "$work/project"
(cd "$work/project" && GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off go test -race -count=1 -buildvcs=false ./...)
for corpus in corpus-a corpus-b; do
  (cd "$work/project" && GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./streamservice -run '^TestNativeTransportRuntimeCorpus$' -args -native-transport-corpus-dir "$work/$corpus")
done
cmp "$work/corpus-a/requests.jsonl" "$work/corpus-b/requests.jsonl"
cmp "$work/corpus-a/expected.jsonl" "$work/corpus-b/expected.jsonl"
cmp "$work/corpus-a/COMPLETE" "$work/corpus-b/COMPLETE"
test "$(wc -l < "$work/corpus-a/requests.jsonl" | tr -d ' ')" -eq 4096
test "$(wc -l < "$work/corpus-a/expected.jsonl" | tr -d ' ')" -eq 4096
printf 'Go UPB-08 fixture: native race and deterministic 4,096-case ordered transport corpus pass\n'
