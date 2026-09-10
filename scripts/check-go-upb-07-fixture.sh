#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb07-fixture.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; export GOCACHE
"$repo/scripts/materialize-go-upb07-fixture.sh" "$work/project"
(cd "$work/project" && GOTOOLCHAIN=local GOPROXY="file://$repo/fixtures/go-upb03-offline-proxy" GOSUMDB=off go test -race -count=1 -buildvcs=false ./...)
(cd "$work/project" && GOTOOLCHAIN=local GOPROXY="file://$repo/fixtures/go-upb03-offline-proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./persistence -run '^TestNativeDurableCorpus$' -args -native-corpus-dir "$work/corpus-a")
(cd "$work/project" && GOTOOLCHAIN=local GOPROXY="file://$repo/fixtures/go-upb03-offline-proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./persistence -run '^TestNativeDurableCorpus$' -args -native-corpus-dir "$work/corpus-b")
cmp "$work/corpus-a/requests.jsonl" "$work/corpus-b/requests.jsonl"
cmp "$work/corpus-a/expected.jsonl" "$work/corpus-b/expected.jsonl"
cmp "$work/corpus-a/COMPLETE" "$work/corpus-b/COMPLETE"
test "$(wc -l < "$work/corpus-a/requests.jsonl" | tr -d ' ')" -eq 2080
test "$(wc -l < "$work/corpus-a/expected.jsonl" | tr -d ' ')" -eq 2080
echo 'Go UPB-07 fixture: native race tests and deterministic 2,080-case durable planner corpus pass'
