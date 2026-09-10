#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb07-fixture.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; export GOCACHE
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/lift" ./cmd/go-upb07-lift && go build -buildvcs=false -o "$work/observe" ./cmd/canonical-observe)
"$repo/scripts/materialize-go-upb07-fixture.sh" "$work/project"
(cd "$work/project" && GOTOOLCHAIN=local GOPROXY="file://$repo/fixtures/go-upb03-offline-proxy" GOSUMDB=off go test -race -count=1 -buildvcs=false ./...)
(cd "$work/project" && GOTOOLCHAIN=local GOPROXY="file://$repo/fixtures/go-upb03-offline-proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./persistence -run '^TestNativeDurableCorpus$' -args -native-corpus-dir "$work/corpus-a")
(cd "$work/project" && GOTOOLCHAIN=local GOPROXY="file://$repo/fixtures/go-upb03-offline-proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./persistence -run '^TestNativeDurableCorpus$' -args -native-corpus-dir "$work/corpus-b")
cmp "$work/corpus-a/requests.jsonl" "$work/corpus-b/requests.jsonl"
cmp "$work/corpus-a/expected.jsonl" "$work/corpus-b/expected.jsonl"
cmp "$work/corpus-a/COMPLETE" "$work/corpus-b/COMPLETE"
test "$(wc -l < "$work/corpus-a/requests.jsonl" | tr -d ' ')" -eq 2080
test "$(wc -l < "$work/corpus-a/expected.jsonl" | tr -d ' ')" -eq 2080
"$work/lift" -project "$work/project" -execution-g1 "$repo/modules/execution/v36/module.g1" -out "$work/program.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/program.g1" "$work/program.seme"
"$work/observe" "$work/program.seme" < "$work/corpus-a/requests.jsonl" > "$work/canonical.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/corpus-a/expected.jsonl" "$work/canonical.jsonl"
echo 'Go UPB-07 fixture: native race, deterministic 2,080-case corpus, lift, and canonical parity pass'
