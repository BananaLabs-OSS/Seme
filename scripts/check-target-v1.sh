#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-target-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

k0="$repo/bootstrap/seme-k0-linux-amd64"
g1="$repo/compiler/g1-compiler.k0"
kernel="$repo/compiler/kernel-wire-validator.k0"
foundation_validator="$repo/modules/foundation/v1/validator.k0"
provider_module="$repo/modules/provider/v1/module.g1"
foundation_module="$repo/modules/foundation/v1/module.g1"
execution_module="$repo/modules/execution/v6/module.g1"
execution="$repo/modules/execution/v6"
package_module="$repo/modules/package/v1/module.g1"
target="$repo/modules/target/v1"

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
plan_check="$work/target-plan-check"
(
    cd "$repo/reference/go"
    go build -o "$go_provider" ./cmd/go-provider
    go build -o "$target_plan" ./cmd/target-plan
    go build -o "$plan_check" ./cmd/target-plan-check
    go run ./cmd/execution-module-v6 > "$work/execution-module.g1"
    go run ./cmd/target-module > "$work/target-module.g1"
)

(
    cd "$execution"
    sha256sum -c module.g1.sha256
    sha256sum -c module.seme.sha256
)
cmp "$execution/module.g1" "$work/execution-module.g1"
"$k0" "$g1" "$work/execution-module.g1" "$work/execution-module.seme"
cmp "$execution/module.seme" "$work/execution-module.seme"
"$k0" "$kernel" "$execution/module.seme"
"$k0" "$foundation_validator" "$execution/module.seme"

(
    cd "$target"
    sha256sum -c module.g1.sha256
    sha256sum -c module.seme.sha256
)
cmp "$target/module.g1" "$work/target-module.g1"
"$k0" "$g1" "$work/target-module.g1" "$work/target-module.seme"
cmp "$target/module.seme" "$work/target-module.seme"
"$k0" "$kernel" "$target/module.seme"
"$k0" "$foundation_validator" "$target/module.seme"

cp -R "$repo/fixtures/go-target-v1" "$work/original"
cp -R "$repo/fixtures/go-target-v1" "$work/project"
(cd "$work/project" && go test ./...)
"$go_provider" ingest --project "$work/project" --module "$provider_module" --out "$work/import"
"$go_provider" ingest --project "$work/project" --module "$provider_module" --out "$work/import-repeat"
cmp "$work/import/manifest.json" "$work/import-repeat/manifest.json"
cmp "$work/import/program.g1" "$work/import-repeat/program.g1"
rg -q '"qualified_name": "example.com/seme-quota-log-proof/internal/policy.WithinLimit"' "$work/import/manifest.json"

emit_plan() {
    project=$1
    manifest=$2
    provider_graph=$3
    policy=$4
    output=$5
    "$target_plan" \
        --project "$project" --manifest "$manifest" \
        --provider-graph "$provider_graph" \
        --foundation-module "$foundation_module" \
        --execution-module "$execution_module" \
        --package-module "$package_module" \
        --target-module "$target/module.g1" \
        --policy "$policy" --out "$output"
}

for policy in allow-adapted exact-only
do
    emit_plan "$work/project" "$work/import/manifest.json" "$work/import/program.g1" "$policy" "$work/$policy-a.g1"
    emit_plan "$work/project" "$work/import/manifest.json" "$work/import/program.g1" "$policy" "$work/$policy-b.g1"
    cmp "$work/$policy-a.g1" "$work/$policy-b.g1"
    "$k0" "$g1" "$work/$policy-a.g1" "$work/$policy.seme"
    "$k0" "$kernel" "$work/$policy.seme"
    "$k0" "$foundation_validator" "$work/$policy.seme"
    "$plan_check" "$work/$policy.seme" "$work/import/manifest.json" "$policy"
done

# Equivalent Go presentation choices must lift through the same semantic
# handlers instead of being tied to one handwritten spelling. Reverse the empty
# comparison, rename the local binding, and change the error payload; the plan
# must preserve the new source literal.
cp -R "$work/project" "$work/generalized"
sed \
    -e 's/request.Subject == ""/"" == request.Subject/' \
    -e 's/accepted :=/decision :=/' \
    -e 's/, accepted)/, decision)/' \
    -e 's/Accepted: accepted/Accepted: decision/' \
    -e 's/subject required/subject missing/g' \
    "$work/project/quota.go" > "$work/generalized/quota.go"
sed 's/subject required/subject missing/g' \
    "$work/project/quota_test.go" > "$work/generalized/quota_test.go"
(cd "$work/generalized" && go test ./...)
"$go_provider" ingest --project "$work/generalized" --module "$provider_module" --out "$work/generalized-import"
emit_plan "$work/generalized" "$work/generalized-import/manifest.json" "$work/generalized-import/program.g1" allow-adapted "$work/generalized.g1"
rg -q '^fi 00000000000000000000000000009500 by 7375626a656374206d697373696e67$' "$work/generalized.g1"
"$k0" "$g1" "$work/generalized.g1" "$work/generalized.seme"
"$k0" "$kernel" "$work/generalized.seme"
"$k0" "$foundation_validator" "$work/generalized.seme"
"$plan_check" "$work/generalized.seme" "$work/generalized-import/manifest.json" allow-adapted

# A native concurrent edit invalidates the provider evidence even when it would
# not change the recognized AST profile.
cp -R "$work/project" "$work/stale"
printf '\n// concurrent native edit\n' >> "$work/stale/quota.go"
expect_status 65 emit_plan "$work/stale" "$work/import/manifest.json" "$work/import/program.g1" allow-adapted "$work/stale.g1"

# Nested package sources participate in the provider revision even before
# Provider Contract v1 projects their declarations. Dependency edits therefore
# cannot reuse stale semantic evidence.
cp -R "$work/project" "$work/stale-dependency"
printf '\n// concurrent dependency edit\n' >> "$work/stale-dependency/internal/policy/policy.go"
expect_status 65 emit_plan "$work/stale-dependency" "$work/import/manifest.json" "$work/import/program.g1" allow-adapted "$work/stale-dependency.g1"

# Re-ingested source with a changed observable message remains valid Go but is
# outside the exact target-analysis profile and must reject rather than receive
# guessed logging semantics.
cp -R "$work/project" "$work/unsupported"
sed 's/quota.accepted=%t/quota.decision=%t/' "$work/project/quota.go" > "$work/unsupported/quota.go"
(cd "$work/unsupported" && go test ./... >/dev/null 2>&1 || true)
"$go_provider" ingest --project "$work/unsupported" --module "$provider_module" --out "$work/unsupported-import"
expect_status 65 emit_plan "$work/unsupported" "$work/unsupported-import/manifest.json" "$work/unsupported-import/program.g1" allow-adapted "$work/unsupported.g1"
expect_status 65 "$target_plan" \
    --project "$work/project" --manifest "$work/import/manifest.json" \
    --provider-graph "$work/import/program.g1" \
    --foundation-module "$foundation_module" --execution-module "$execution_module" \
    --package-module "$package_module" --target-module "$target/module.g1" \
    --policy dishonest-exact --out "$work/dishonest.g1"

cmp "$work/original/go.mod" "$work/project/go.mod"
cmp "$work/original/quota.go" "$work/project/quota.go"
cmp "$work/original/quota_test.go" "$work/project/quota_test.go"
cmp "$work/original/internal/policy/policy.go" "$work/project/internal/policy/policy.go"
cmp "$work/original/README.md" "$work/project/README.md"

echo "Target Contract v1: generalized Go profile analysis and honest Wasm/Pulp planning passed"
