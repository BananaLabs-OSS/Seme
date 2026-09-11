#!/bin/sh
# Focused JavaScript identity-bound project reconciliation proof. Only the
# cumulative UPB-11 gate may claim the scorecard cell.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
fixture="$repo/fixtures/javascript-upb05-configuration"
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-upb11-reconcile.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
target=$(cd "$repo/reference/js" && node --input-type=module -e 'import{javascriptDeclarationIdentity as id}from"./javascript-provider.mjs";process.stdout.write(id("example.test/javascript-upb05","InitializePolicy"))')

run() {
  node "$repo/reference/js/javascript-project-reconcile-cli.mjs" \
    --project "$fixture" --out "$1" \
    --project-path example.test/javascript-upb05 \
    --files application.js,configuration.js,controlled.js,policy.js,state.js,transport.js \
    --module "$repo/modules/execution/v36/module.g1" --entry Run \
    --target "$2" --expected InitializePolicy --replacement "$3" --revision 2 \
    --native-runner "$repo/reference/js/javascript-upb11-native-runner.mjs"
}

run "$work/a" "$target" BuildPolicy
run "$work/b" "$target" BuildPolicy
diff -ru "$work/a" "$work/b"
rg -q 'export function BuildPolicy' "$work/a/policy.js"
rg -q 'import \{ BuildPolicy \}' "$work/a/application.js"
rg -q 'InitializePolicy is historical prose' "$work/a/policy.js"
test -s "$work/a/.seme-reconciliation-v1/prior-provider.g1"
test -s "$work/a/.seme-reconciliation-v1/result-provider.g1"
test -s "$work/a/.seme-reconciliation-v1/native-validation.txt"
test -s "$work/a/.seme-reconciliation-v1/COMPLETE.sha256"
verify() {
  node "$repo/reference/js/javascript-project-reconcile-verify-cli.mjs" \
    --project "$1" --project-path example.test/javascript-upb05 \
    --files application.js,configuration.js,controlled.js,policy.js,state.js,transport.js \
    --module "$repo/modules/execution/v36/module.g1" --entry Run \
    --native-runner "$repo/reference/js/javascript-upb11-native-runner.mjs"
}
verify "$work/a" > "$work/report.json"
rg -q '"replacement": "BuildPolicy"' "$work/report.json"

reject() {
  name=$1; identity=$2; replacement=$3
  if run "$work/reject-$name" "$identity" "$replacement" >"$work/$name.out" 2>"$work/$name.err"; then
    echo "JavaScript UPB-11 accepted $name" >&2; exit 1
  fi
  test ! -e "$work/reject-$name"
  test ! -s "$work/$name.out"
}
reject forged 00000000000000000000000000000000 BuildPolicy
reject collision "$target" Run
reject keyword "$target" class
if run "$work/a" "$target" BuildPolicy >"$work/existing.out" 2>"$work/existing.err"; then
  echo 'JavaScript UPB-11 overwrote an existing destination' >&2; exit 1
fi
test ! -s "$work/existing.out"

cp -R "$work/a" "$work/tampered"
printf x >> "$work/tampered/.seme-reconciliation-v1/result-provider.g1"
if verify "$work/tampered" >"$work/tampered.out" 2>"$work/tampered.err"; then echo 'JavaScript UPB-11 accepted metadata tamper' >&2;exit 1;fi
test ! -s "$work/tampered.out"

printf 'JavaScript UPB-11 focused reconciliation: stable identity, declaration/import projection, native parity, deterministic publication, and atomic adversaries pass\n'
