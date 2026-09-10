#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb05-fixture.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; export GOCACHE
"$repo/scripts/materialize-go-upb05-fixture.sh" "$work/project"
(cd "$work/project" && GOTOOLCHAIN=local GOPROXY="file://$repo/fixtures/go-upb03-offline-proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./...)
(cd "$work/project" && GOTOOLCHAIN=local GOPROXY="file://$repo/fixtures/go-upb03-offline-proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./service -run '^TestNativeConfigurationCorpus$' -args -native-corpus-dir "$work/corpus-a")
(cd "$work/project" && GOTOOLCHAIN=local GOPROXY="file://$repo/fixtures/go-upb03-offline-proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./service -run '^TestNativeConfigurationCorpus$' -args -native-corpus-dir "$work/corpus-b")
cmp "$work/corpus-a/requests.jsonl" "$work/corpus-b/requests.jsonl"
cmp "$work/corpus-a/expected.jsonl" "$work/corpus-b/expected.jsonl"
cmp "$work/corpus-a/COMPLETE" "$work/corpus-b/COMPLETE"
test "$(wc -l < "$work/corpus-a/requests.jsonl" | tr -d ' ')" -eq 2058
test "$(wc -l < "$work/corpus-a/expected.jsonl" | tr -d ' ')" -eq 2058
if rg -n 'os\.Getenv|os\.LookupEnv|func init\(|\btime\.|\brand\.|^var [A-Za-z_][A-Za-z0-9_]* =' "$work/project" --glob '*.go' --glob '!**/*_test.go'; then
  echo 'Go UPB-05 fixture contains an ambient or global initialization mechanism' >&2
  exit 1
fi
echo 'Go UPB-05 fixture: cumulative 2,048 corpus and explicit configuration/lifecycle matrix pass'
