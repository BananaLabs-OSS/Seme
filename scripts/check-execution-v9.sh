#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v9.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

k0="$repo/bootstrap/seme-k0-linux-amd64"
g1="$repo/compiler/g1-compiler.k0"
kernel="$repo/compiler/kernel-wire-validator.k0"
foundation="$repo/modules/foundation/v1/validator.k0"
execution="$repo/modules/execution/v9"

(
    cd "$repo/reference/go"
    go test ./executionmodule ./goprovider ./wasmtarget
    go run ./cmd/execution-module-v9 > "$work/module.g1"
    go build -o "$work/go-provider" ./cmd/go-provider
    go build -o "$work/go-lift" ./cmd/go-execution-lift
    go build -o "$work/structured-check" ./cmd/structured-v9-check
)

cmp "$execution/module.g1" "$work/module.g1"
(
    cd "$execution"
    sha256sum -c module.g1.sha256
    sha256sum -c module.seme.sha256
)
"$k0" "$g1" "$execution/module.g1" "$work/module.seme"
cmp "$execution/module.seme" "$work/module.seme"
"$k0" "$kernel" "$execution/module.seme"
"$k0" "$foundation" "$execution/module.seme"

cp -R "$repo/fixtures/go-execution-v9" "$work/project"
(cd "$work/project" && go test ./...)
"$work/go-provider" ingest \
    --project "$work/project" --module "$repo/modules/provider/v1/module.g1" \
    --out "$work/import"
"$work/go-lift" \
    --project "$work/project" --manifest "$work/import/manifest.json" \
    --module "$execution/module.g1" --function MultiplyNested \
    --profile structured-v8 --out "$work/program-a.g1"
"$work/go-lift" \
    --project "$work/project" --manifest "$work/import/manifest.json" \
    --module "$execution/module.g1" --function MultiplyNested \
    --profile structured-v8 --out "$work/program-b.g1"
cmp "$work/program-a.g1" "$work/program-b.g1"
"$k0" "$g1" "$work/program-a.g1" "$work/program.seme"
"$k0" "$kernel" "$work/program.seme"
"$k0" "$foundation" "$work/program.seme"
"$work/structured-check" "$work/program.seme"

version=2
while [ "$version" -le 8 ]; do
    (cd "$repo/reference/go" && go run "./cmd/execution-module-v$version") > "$work/module-v$version.g1"
    cmp "$repo/modules/execution/v$version/module.g1" "$work/module-v$version.g1"
    version=$((version + 1))
done

echo "Core Execution v9: compositional integer multiplication passed"
