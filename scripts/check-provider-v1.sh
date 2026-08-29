#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-provider-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

k0="$repo/bootstrap/seme-k0-linux-amd64"
g1="$repo/compiler/g1-compiler.k0"
kernel="$repo/compiler/kernel-wire-validator.k0"
foundation="$repo/modules/foundation/v1/validator.k0"
provider="$repo/modules/provider/v1"
patch="$repo/modules/patch/v1"
go_provider="$work/go-provider"

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

(
    cd "$provider"
    sha256sum -c module.g1.sha256
    sha256sum -c module.seme.sha256
)
(cd "$repo/reference/go" && go run ./cmd/provider-module) > "$work/provider.g1"
(cd "$repo/reference/go" && go build -o "$go_provider" ./cmd/go-provider)
cmp "$provider/module.g1" "$work/provider.g1"
"$k0" "$g1" "$work/provider.g1" "$work/provider.seme"
cmp "$provider/module.seme" "$work/provider.seme"
"$k0" "$kernel" "$work/provider.seme"
"$k0" "$foundation" "$work/provider.seme"

cp -R "$repo/fixtures/go-provider-v1" "$work/original"
cp -R "$repo/fixtures/go-provider-v1" "$work/project"
(cd "$work/project" && go test ./...)

# First ingestion is deterministic without any prior identity evidence.
"$go_provider" ingest \
    --project "$work/project" --module "$provider/module.g1" \
    --out "$work/import-a"
"$go_provider" ingest \
    --project "$work/project" --module "$provider/module.g1" \
    --out "$work/import-a-repeat"
cmp "$work/import-a/manifest.json" "$work/import-a-repeat/manifest.json"
cmp "$work/import-a/program.g1" "$work/import-a-repeat/program.g1"
"$k0" "$g1" "$work/import-a/program.g1" "$work/import-a/program.seme"
"$k0" "$kernel" "$work/import-a/program.seme"
"$k0" "$foundation" "$work/import-a/program.seme"

target=$(awk '
    /"id":/ { id = $2; gsub(/[",]/, "", id) }
    /"name": "Greeting"/ { print id }
' "$work/import-a/manifest.json")
if [ -z "$target" ]; then
    echo "Greeting semantic identity not found" >&2
    exit 1
fi

# Compose and execute the edit through canonical Patch Module v1.
(cd "$repo/reference/go" && go run ./cmd/provider-patch-fixture \
    "$patch/module.g1" "$work/import-a/program.g1" "$target" \
    Greeting Welcome) > "$work/patch.g1"
"$k0" "$g1" "$work/patch.g1" "$work/patch.seme"
"$k0" "$kernel" "$work/patch.seme"
"$k0" "$foundation" "$work/patch.seme"
"$repo/scripts/apply-patch-v1.sh" "$work/patch.seme" "$work/patched.seme"
"$k0" "$kernel" "$work/patched.seme"
"$k0" "$foundation" "$work/patched.seme"

# An unstamped semantic candidate and a native concurrent edit both fail before
# touching the ordinary project.
cp "$work/project/greet.go" "$work/greet.before-failure"
expect_status 65 "$go_provider" rename \
    --project "$work/project" --manifest "$work/import-a/manifest.json" \
    --target "$target" --base "$work/patch.seme" \
    --candidate "$work/patch.seme" --report "$work/uncommitted-report.json"
cmp "$work/greet.before-failure" "$work/project/greet.go"

cp -R "$work/project" "$work/stale-project"
printf '\n// concurrent native edit\n' >> "$work/stale-project/greet.go"
cp "$work/stale-project/greet.go" "$work/stale.before"
expect_status 65 "$go_provider" rename \
    --project "$work/stale-project" --manifest "$work/import-a/manifest.json" \
    --target "$target" --base "$work/patch.seme" \
    --candidate "$work/patched.seme" --report "$work/stale-report.json"
cmp "$work/stale.before" "$work/stale-project/greet.go"

"$go_provider" rename \
    --project "$work/project" --manifest "$work/import-a/manifest.json" \
    --target "$target" --base "$work/patch.seme" \
    --candidate "$work/patched.seme" --report "$work/projection.json" \
    --validate
(cd "$work/project" && go test ./...)

# Re-ingestion must use old evidence, recover identities, and remain fully
# canonical. Repeating it proves reconciliation determinism.
"$go_provider" ingest \
    --project "$work/project" --module "$provider/module.g1" \
    --prior "$work/import-a/manifest.json" --out "$work/import-b"
"$go_provider" ingest \
    --project "$work/project" --module "$provider/module.g1" \
    --prior "$work/import-a/manifest.json" --out "$work/import-b-repeat"
cmp "$work/import-b/manifest.json" "$work/import-b-repeat/manifest.json"
cmp "$work/import-b/program.g1" "$work/import-b-repeat/program.g1"
"$k0" "$g1" "$work/import-b/program.g1" "$work/import-b/program.seme"
"$k0" "$kernel" "$work/import-b/program.seme"
"$k0" "$foundation" "$work/import-b/program.seme"
(cd "$repo/reference/go" && go run ./cmd/provider-report \
    "$provider/module.g1" "$work/projection.json" \
    "$work/import-b/manifest.json") > "$work/projection.g1"
"$k0" "$g1" "$work/projection.g1" "$work/projection.seme"
"$k0" "$kernel" "$work/projection.seme"
"$k0" "$foundation" "$work/projection.seme"
(cd "$repo/reference/go" && go run ./cmd/provider-roundtrip-check \
    "$work/import-a/manifest.json" "$work/import-b/manifest.json" \
    "$work/projection.json" "$work/original" "$work/project" "$target")

cmp "$work/original/go.mod" "$work/project/go.mod"
cmp "$work/original/greet_test.go" "$work/project/greet_test.go"
cmp "$work/original/README.md" "$work/project/README.md"
(cd "$repo/reference/go" && go test ./...)

echo "Provider Contract v1: canonical Go import, Patch rename, minimal projection, native validation, and identity-preserving re-import passed"
