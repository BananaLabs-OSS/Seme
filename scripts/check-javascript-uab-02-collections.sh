#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-uab02-collections.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; GOCACHE="$work/go-cache"; export XDG_CACHE_HOME GOCACHE

node "$repo/reference/js/javascript-provider-cli.mjs" --source "$repo/fixtures/javascript-uab-02/collections.js" \
  --module "$repo/modules/execution/v32/module.g1" --package example.test/javascript-uab-02-collections \
  --revision 1 --out "$work/source.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/source.g1" "$work/program.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/source.g1" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v32/module.g1" --package example.test/javascript-uab-02-collections \
  --revision 1 --out "$work/relifted.g1"
cmp "$work/source.g1" "$work/relifted.g1"

vectors="$repo/fixtures/javascript-uab-02/collection-vectors.json"
node "$repo/reference/js/javascript-aggregate-native-runner.mjs" "file://$repo/fixtures/javascript-uab-02/collections.js" "$vectors" > "$work/native.json"
node "$repo/reference/js/javascript-aggregate-native-runner.mjs" "file://$work/projected.mjs" "$vectors" > "$work/projected.json"
node "$repo/reference/js/json-equal.mjs" "$work/native.json" "$work/projected.json"
if [ -n "${SEME_CANONICAL_EVAL:-}" ]; then cp "$SEME_CANONICAL_EVAL" "$work/canonical-eval"; else (cd "$repo/reference/go" && go build -buildvcs=false -o "$work/canonical-eval" ./cmd/canonical-eval); fi
node "$repo/reference/js/canonical-vector-adapter.mjs" build collections "$vectors" > "$work/canonical-vectors.json"
"$work/canonical-eval" "$work/program.seme" "$work/canonical-vectors.json" > "$work/canonical-values.json"
node "$repo/reference/js/canonical-vector-adapter.mjs" normalize collections "$work/canonical-values.json" > "$work/canonical.json"
node "$repo/reference/js/json-equal.mjs" "$work/native.json" "$work/canonical.json"

if [ -n "${SEME_AGGREGATE_WASM_LOWER:-}" ]; then cp "$SEME_AGGREGATE_WASM_LOWER" "$work/lower"; else (cd "$repo/reference/go" && go build -buildvcs=false -o "$work/lower" ./cmd/aggregate-wasm-lower); fi
"$work/lower" "$work/program.seme" "$work/program.wasm" "$work/abi.json"
node "$repo/reference/js/aggregate-function-runner.mjs" "$work/program.wasm" "$vectors" > "$work/wasm.json"
node "$repo/reference/js/json-equal.mjs" "$work/native.json" "$work/wasm.json"
node "$repo/reference/js/aggregate-function-runner.mjs" "$work/program.wasm" "$vectors" --tsv > "$work/vectors.tsv"

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-aggregate-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/program.wasm" "$work/pulp/pure-function.wasm"
while IFS='	' read -r kind name request response; do
  if [ "$kind" = valid ]; then
    (cd "$work/pulp" && "$work/pulp-runner" -manifest pulp.cell.toml -provider seme.function-aggregate-v1 -request "$request") > "$work/pulp-$name.log" 2>&1
    rg -q "\"response\":\"$response\"" "$work/pulp-$name.log"
  elif (cd "$work/pulp" && "$work/pulp-runner" -manifest pulp.cell.toml -provider seme.function-aggregate-v1 -request "$request") > "$work/pulp-malformed-$name.log" 2>&1; then
    echo "Pulp accepted malformed aggregate vector $name" >&2; exit 1
  fi
done < "$work/vectors.tsv"

echo "JavaScript UAB-02 collections: identical original/projected/Wasm/Pulp vectors and structural rejection pass"
