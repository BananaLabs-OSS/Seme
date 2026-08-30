#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-cli-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

cp -R "$repo/fixtures/go-target-v1" "$work/project"
cp -R "$work/project" "$work/original"

"$repo/seme" doctor > "$work/doctor.txt"
"$repo/seme" build "$work/project" "$work/output/application.wasm" > "$work/build.txt"

test -s "$work/output/application.wasm"
test -s "$work/project/.seme/manifest.json"
test -s "$work/project/.seme/provider.g1"
test -s "$work/project/.seme/plan.g1"
test -s "$work/project/.seme/plan.seme"
test -s "$work/project/.seme/report.json"
rg -q '"status": "executable"' "$work/project/.seme/report.json"
rg -q '"fidelity": "adapted"' "$work/project/.seme/report.json"
rg -q 'Fidelity: adapted' "$work/build.txt"
"$repo/seme" inspect "$work/project" > "$work/inspect.json"
cmp "$work/project/.seme/report.json" "$work/inspect.json"

node "$repo/reference/js/wasm-target-runner.mjs" \
    "$work/output/application.wasm" 40 2 50 tenant-a > "$work/result.json"
rg -q '"result":true' "$work/result.json"
rg -q '"events":\[true\]' "$work/result.json"

cmp "$work/original/go.mod" "$work/project/go.mod"
cmp "$work/original/quota.go" "$work/project/quota.go"
cmp "$work/original/quota_test.go" "$work/project/quota_test.go"
cmp "$work/original/internal/policy/policy.go" "$work/project/internal/policy/policy.go"

cp -R "$work/original" "$work/unsupported"
sed 's/+ 0/* 1/' \
    "$work/original/internal/policy/policy.go" > "$work/unsupported/internal/policy/policy.go"
set +e
"$repo/seme" build "$work/unsupported" "$work/unsupported.wasm" \
    > "$work/unsupported.out" 2> "$work/unsupported.err"
status=$?
set -e
test "$status" -eq 65
rg -q 'supported helper expressions currently use' "$work/unsupported.err"
test ! -e "$work/unsupported.wasm"

echo "Seme CLI v1: one-command build, evidence, execution, preservation, and actionable rejection passed"
