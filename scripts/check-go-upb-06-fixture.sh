#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT HUP INT TERM
project="$work/project"
corpus_a="$work/corpus-a"
corpus_b="$work/corpus-b"
"$repo/scripts/materialize-go-upb06-fixture.sh" "$project"
(cd "$project" && GOWORK=off GOPROXY=off go test ./...)
(cd "$project" && GOWORK=off GOPROXY=off go test ./service -run '^TestNativeResourceCorpus$' -native-resource-corpus-dir "$corpus_a")
(cd "$project" && GOWORK=off GOPROXY=off go test ./service -run '^TestNativeResourceCorpus$' -native-resource-corpus-dir "$corpus_b")
cmp "$corpus_a/requests.jsonl" "$corpus_b/requests.jsonl"
cmp "$corpus_a/expected.jsonl" "$corpus_b/expected.jsonl"
cmp "$corpus_a/COMPLETE" "$corpus_b/COMPLETE"
test "$(wc -l < "$corpus_a/requests.jsonl")" -eq 2066
test "$(wc -l < "$corpus_a/expected.jsonl")" -eq 2066
test "$(cat "$corpus_a/COMPLETE")" = "seme-go-upb06-native-v1
2066"
test "$(wc -c < "$project/resources/notice.txt")" -eq 26
test "$(wc -c < "$project/resources/marker.bin")" -eq 7
test "$(sha256sum "$project/resources/notice.txt" | cut -d ' ' -f 1)" = dde7c6f27c3a259a48b5c9e6b886a7f3c1c75f87c73699536eccb7149d1e7d48
test "$(sha256sum "$project/resources/marker.bin" | cut -d ' ' -f 1)" = d57b9b19e900f27112610f80812bf55d383d69ffd0e5796b7e1c84164ef15f9d
grep -R -n -E 'os\.Getenv|os\.LookupEnv|func init\(|time\.(Now|Sleep)|math/rand|crypto/rand' "$project" --include='*.go' --exclude='*_test.go' && exit 1 || true
printf 'go UPB-06 ordinary resource fixture: pass (2,066 deterministic native observations)\n'
