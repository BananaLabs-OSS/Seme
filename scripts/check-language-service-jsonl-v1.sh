#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-language-jsonl.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

(
    cd "$repo/reference/go"
    go test -buildvcs=false ./cmd/language-service-jsonl -count=1
    go build -buildvcs=false -o "$work/language-service-jsonl" \
        ./cmd/language-service-jsonl
)

printf '%s\n' \
    '{"id":"init","command":"initialize","session":"one","package_path":"example.test/stream"}' \
    '{"id":"valid","command":"update","session":"one","revision":1,"files":{"main.go":"package stream\nfunc Decide(enabled bool, value, limit int64) bool {\nif enabled || (\"λ\" + \"!\" == \"never\") {\nif value <= limit { return \"exact\" + \"-text\" == \"exact-text\" }\nreturn false\n}\nreturn false\n}\n"}}' \
    '{"id":"invalid","command":"update","session":"one","revision":2,"files":{"main.go":"package stream\nfunc Decide("}}' \
    '{"id":"stale","command":"update","session":"one","revision":2,"files":{"main.go":"package stream\nfunc Decide(enabled bool, value, limit int64) bool { return enabled && value <= limit }\n"}}' \
    '{"id":"snapshot","command":"snapshot","session":"one"}' \
    | "$work/language-service-jsonl" \
        --module "$repo/modules/execution/v13/module.g1" > "$work/first.jsonl"

printf '%s\n' \
    '{"id":"init","command":"initialize","session":"one","package_path":"example.test/stream"}' \
    '{"id":"valid","command":"update","session":"one","revision":1,"files":{"main.go":"package stream\nfunc Decide(enabled bool, value, limit int64) bool {\nif enabled || (\"λ\" + \"!\" == \"never\") {\nif value <= limit { return \"exact\" + \"-text\" == \"exact-text\" }\nreturn false\n}\nreturn false\n}\n"}}' \
    '{"id":"invalid","command":"update","session":"one","revision":2,"files":{"main.go":"package stream\nfunc Decide("}}' \
    '{"id":"stale","command":"update","session":"one","revision":2,"files":{"main.go":"package stream\nfunc Decide(enabled bool, value, limit int64) bool { return enabled && value <= limit }\n"}}' \
    '{"id":"snapshot","command":"snapshot","session":"one"}' \
    | "$work/language-service-jsonl" \
        --module "$repo/modules/execution/v13/module.g1" > "$work/second.jsonl"

cmp "$work/first.jsonl" "$work/second.jsonl"
grep -q '"id":"valid".*"disposition":"accepted-valid"' "$work/first.jsonl"
grep -q '"id":"invalid".*"last_valid_revision":1.*"disposition":"accepted-invalid"' "$work/first.jsonl"
grep -q '"id":"stale".*"accepted":false.*"last_valid_revision":1.*"disposition":"rejected-stale"' "$work/first.jsonl"
grep -q '"id":"snapshot".*"disposition":"rejected-stale"' "$work/first.jsonl"

echo "Language service JSON-lines v1: deterministic live update, retention, stale rejection, framing, and EOF passed"
