#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-application-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

k0="$repo/bootstrap/seme-k0-linux-amd64"
g1="$repo/compiler/g1-compiler.k0"
kernel="$repo/compiler/kernel-wire-validator.k0"
foundation="$repo/modules/foundation/v1/validator.k0"
lowerer="$repo/compiler/k0-module-lowerer.k0"
execution="$repo/modules/execution/v2"
package="$repo/modules/package/v1"
interpreter="$repo/modules/execution/v1"

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
go_lift="$work/go-execution-lift"
vectors="$work/execution-vector"
contract_check="$work/application-contract-check"
(
    cd "$repo/reference/go"
    go build -o "$go_provider" ./cmd/go-provider
    go build -o "$go_lift" ./cmd/go-execution-lift
    go build -o "$vectors" ./cmd/execution-vector
    go build -o "$contract_check" ./cmd/application-contract-check
    go run ./cmd/execution-module-v2 > "$work/execution-module.g1"
    go run ./cmd/package-module > "$work/package-module.g1"
)

for module in "$execution" "$package"
do
    (cd "$module" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
    "$k0" "$kernel" "$module/module.seme"
    "$k0" "$foundation" "$module/module.seme"
done
cmp "$execution/module.g1" "$work/execution-module.g1"
cmp "$package/module.g1" "$work/package-module.g1"
"$k0" "$g1" "$work/execution-module.g1" "$work/execution-module.seme"
"$k0" "$g1" "$work/package-module.g1" "$work/package-module.seme"
cmp "$execution/module.seme" "$work/execution-module.seme"
cmp "$package/module.seme" "$work/package-module.seme"

(cd "$interpreter" && sha256sum -c interpreter.k0.sha256 && sha256sum -c interpreter.g1.sha256 && sha256sum -c interpreter.seme.sha256)
"$k0" "$repo/bootstrap/s1-compiler.k0" "$interpreter/interpreter.s1" "$work/interpreter.k0"
cmp "$interpreter/interpreter.k0" "$work/interpreter.k0"
"$k0" "$lowerer" "$interpreter/interpreter.seme" "$work/interpreter-lowered.k0"
cmp "$interpreter/interpreter.k0" "$work/interpreter-lowered.k0"
"$k0" "$kernel" "$interpreter/interpreter.seme"

cp -R "$repo/fixtures/go-application-v1" "$work/original"
cp -R "$repo/fixtures/go-application-v1" "$work/project"
(cd "$work/project" && go test ./...)
"$go_provider" ingest \
    --project "$work/project" --module "$repo/modules/provider/v1/module.g1" \
    --out "$work/import"
"$go_lift" \
    --project "$work/project" --manifest "$work/import/manifest.json" \
    --module "$execution/module.g1" --package-module "$package/module.g1" \
    --function Admit --profile quota-v1 --out "$work/program-a.g1"
"$go_lift" \
    --project "$work/project" --manifest "$work/import/manifest.json" \
    --module "$execution/module.g1" --package-module "$package/module.g1" \
    --function Admit --profile quota-v1 --out "$work/program-b.g1"
cmp "$work/program-a.g1" "$work/program-b.g1"
"$k0" "$g1" "$work/program-a.g1" "$work/program.seme"
"$k0" "$kernel" "$work/program.seme"
"$k0" "$foundation" "$work/program.seme"
"$contract_check" "$work/program.seme" "$work/import/manifest.json"

index=0
for values in \
    "40 2 50" \
    "40 20 50" \
    "-10 3 -5" \
    "9223372036854775807 1 0" \
    "0 0 0"
do
    set -- $values
    args="$work/args-$index.bin"
    result="$work/result-$index.bin"
    "$vectors" write-quota "$1" "$2" "$3" "$args"
    "$k0" "$interpreter/interpreter.k0" "$work/program.seme" "$args" "$result"
    "$vectors" check-quota "$1" "$2" "$3" "$result"
    index=$((index + 1))
done

# Source outside the exact profile, missing package evidence, and malformed
# runtime arguments all fail closed.
cp -R "$work/project" "$work/unsupported"
sed 's/<= limit/< limit/' "$work/project/quota.go" > "$work/unsupported/quota.go"
expect_status 65 "$go_lift" \
    --project "$work/unsupported" --manifest "$work/import/manifest.json" \
    --module "$execution/module.g1" --package-module "$package/module.g1" \
    --function Admit --profile quota-v1 --out "$work/unsupported.g1"
"$go_lift" \
    --project "$work/project" --manifest "$work/import/manifest.json" \
    --module "$execution/module.g1" --function Admit --profile quota-v1 \
    --out "$work/no-package.g1"
"$k0" "$g1" "$work/no-package.g1" "$work/no-package.seme"
expect_status 65 "$contract_check" "$work/no-package.seme" "$work/import/manifest.json"
printf '\000' > "$work/bad-args.bin"
expect_status 65 "$k0" "$interpreter/interpreter.k0" \
    "$work/program.seme" "$work/bad-args.bin" "$work/bad-result.bin"

cmp "$work/original/go.mod" "$work/project/go.mod"
cmp "$work/original/quota.go" "$work/project/quota.go"
cmp "$work/original/quota_test.go" "$work/project/quota_test.go"
cmp "$work/original/README.md" "$work/project/README.md"

echo "Package/Application v1: ordinary Go quota policy, explicit contract, and independent Seme execution passed"
