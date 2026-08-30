#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-wasm-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

k0="$repo/bootstrap/seme-k0-linux-amd64"
g1="$repo/compiler/g1-compiler.k0"
kernel="$repo/compiler/kernel-wire-validator.k0"
foundation_validator="$repo/modules/foundation/v1/validator.k0"
artifact="$repo/targets/wasm/pulp-v1/quota-admit.wasm"

expect_status() {
    expected=$1
    shift
    set +e
    "$@"
    actual=$?
    set -e
    if [ "$actual" -ne "$expected" ]; then
        echo "expected status $expected, got $actual: $*" >&2
        exit 1
    fi
}

go_provider="$work/go-provider"
target_plan="$work/target-plan"
wasm_lower="$work/wasm-lower"
result_check="$work/wasm-result-check"
error_check="$work/wasm-error-check"
(
    cd "$repo/reference/go"
    go build -o "$go_provider" ./cmd/go-provider
    go build -o "$target_plan" ./cmd/target-plan
    go build -o "$wasm_lower" ./cmd/wasm-lower
    go build -o "$result_check" ./cmd/wasm-result-check
    go build -o "$error_check" ./cmd/wasm-error-check
)

cp -R "$repo/fixtures/go-target-v1" "$work/original"
cp -R "$repo/fixtures/go-target-v1" "$work/project"
(cd "$work/project" && go test ./...)
"$go_provider" ingest \
    --project "$work/project" --module "$repo/modules/provider/v1/module.g1" \
    --out "$work/import"

emit_plan() {
    policy=$1
    output=$2
    "$target_plan" \
        --project "$work/project" --manifest "$work/import/manifest.json" \
        --provider-graph "$work/import/program.g1" \
        --foundation-module "$repo/modules/foundation/v1/module.g1" \
        --execution-module "$repo/modules/execution/v5/module.g1" \
        --package-module "$repo/modules/package/v1/module.g1" \
        --target-module "$repo/modules/target/v1/module.g1" \
        --policy "$policy" --out "$output"
}

emit_plan allow-adapted "$work/allow.g1"
"$k0" "$g1" "$work/allow.g1" "$work/allow.seme"
"$k0" "$kernel" "$work/allow.seme"
"$k0" "$foundation_validator" "$work/allow.seme"
"$wasm_lower" "$work/allow.seme" "$work/quota-a.wasm"
"$wasm_lower" "$work/allow.seme" "$work/quota-b.wasm"
cmp "$work/quota-a.wasm" "$work/quota-b.wasm"
cmp "$artifact" "$work/quota-a.wasm"
(cd "$repo/targets/wasm/pulp-v1" && sha256sum -c quota-admit.wasm.sha256)

# Runtime phase contains Node's Wasm engine, the derived module, the declared
# host adapter, and scalar arguments. It contains no Go source, executable, AST,
# toolchain, or Go runtime.
index=0
for values in \
    "40 2 50" \
    "40 20 50" \
    "-10 3 -5" \
    "9223372036854775807 1 0" \
    "0 0 0"
do
    set -- $values
    result="$work/result-$index.json"
    node "$repo/reference/js/wasm-target-runner.mjs" \
        "$work/quota-a.wasm" "$1" "$2" "$3" > "$result"
    "$result_check" "$1" "$2" "$3" "$result"
    index=$((index + 1))
done

node "$repo/reference/js/wasm-target-runner.mjs" \
    "$work/quota-a.wasm" 40 2 50 "" > "$work/error.json"
"$error_check" "$work/error.json" "subject required"
node "$repo/reference/js/wasm-malformed-check.mjs" "$work/quota-a.wasm"

# Prove the backend consumes the lifted graph rather than embedding the fixture
# message: change the ordinary Go error text, re-ingest, re-plan, lower, and
# execute the resulting ResultError.
cp -R "$work/project" "$work/message-project"
sed 's/subject required/subject missing/g' \
    "$work/project/quota.go" > "$work/message-project/quota.go"
sed 's/subject required/subject missing/g' \
    "$work/project/quota_test.go" > "$work/message-project/quota_test.go"
(cd "$work/message-project" && go test ./...)
"$go_provider" ingest \
    --project "$work/message-project" --module "$repo/modules/provider/v1/module.g1" \
    --out "$work/message-import"
"$target_plan" \
    --project "$work/message-project" --manifest "$work/message-import/manifest.json" \
    --provider-graph "$work/message-import/program.g1" \
    --foundation-module "$repo/modules/foundation/v1/module.g1" \
    --execution-module "$repo/modules/execution/v5/module.g1" \
    --package-module "$repo/modules/package/v1/module.g1" \
    --target-module "$repo/modules/target/v1/module.g1" \
    --policy allow-adapted --out "$work/message.g1"
"$k0" "$g1" "$work/message.g1" "$work/message.seme"
"$wasm_lower" "$work/message.seme" "$work/message.wasm"
node "$repo/reference/js/wasm-target-runner.mjs" \
    "$work/message.wasm" 40 2 50 "" > "$work/message-error.json"
"$error_check" "$work/message-error.json" "subject missing"

# A valid exact-only plan is deliberately non-executable and cannot be lowered
# by bypassing target policy.
emit_plan exact-only "$work/exact.g1"
"$k0" "$g1" "$work/exact.g1" "$work/exact.seme"
"$k0" "$kernel" "$work/exact.seme"
"$k0" "$foundation_validator" "$work/exact.seme"
expect_status 65 "$wasm_lower" "$work/exact.seme" "$work/forbidden.wasm"

cmp "$work/original/go.mod" "$work/project/go.mod"
cmp "$work/original/quota.go" "$work/project/quota.go"
cmp "$work/original/quota_test.go" "$work/project/quota_test.go"
cmp "$work/original/README.md" "$work/project/README.md"

echo "Wasm target v1: source-derived Result variants lowered and executed with adapted logging effect"
