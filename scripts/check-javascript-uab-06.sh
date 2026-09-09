#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd); pulp_repo=${PULP_REPO:-"$repo/../Pulp"}; pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-js06-full.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
source="$repo/fixtures/javascript-uab-06/program.js"; vectors="$repo/fixtures/javascript-uab-06/vectors.json"; module="$repo/modules/execution/v29/module.g1"
node "$repo/reference/js/javascript-uab-06-native-runner.mjs" "$source" "$vectors" > "$work/native.json"
(cd "$repo/reference/go" && go test -buildvcs=false ./canonicaleval ./wasmtarget && go build -buildvcs=false -o "$work/eval" ./cmd/canonical-eval && go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
test -f "$pulp_repo/go.mod"; git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"; mkdir "$work/pulp-pinned"; git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"; (cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
for pair in Immutable:immutable Mutable:mutable; do
  entry=${pair%%:*}; family=${pair#*:}
  node "$repo/reference/js/javascript-provider-cli.mjs" --source "$source" --module "$module" --package example.test/javascript-uab-06 --revision 1 --entry "$entry" --out "$work/$family.g1"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/$family.g1" "$work/$family.seme"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/$family.seme"
  node "$repo/reference/js/javascript-projector-cli.mjs" "$work/$family.g1" "$work/$family.js"
  node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/$family.js" --module "$module" --package example.test/javascript-uab-06 --revision 1 --entry "$entry" --out "$work/$family-relift.g1"; cmp "$work/$family.g1" "$work/$family-relift.g1"
  node "$repo/reference/js/javascript-uab-06-native-runner.mjs" "$work/$family.js" "$vectors" "$family" > "$work/$family-projected.json"
  node "$repo/reference/js/javascript-uab-06-vectors.mjs" "$vectors" "$family" > "$work/$family-canonical-vectors.json"; "$work/eval" "$work/$family.seme" "$work/$family-canonical-vectors.json" --named-rejections > "$work/$family-canonical.json"
  "$work/lower" "$work/$family.seme" "$work/$family.wasm" "$work/$family-abi.json"; node "$repo/reference/js/javascript-uab-06-wasm-runner.mjs" "$work/$family.wasm" "$vectors" "$family" > "$work/$family-wasm.json"
  node -e 'const fs=require("fs"),u=require("util"),all=JSON.parse(fs.readFileSync(process.argv[1])).valid,p=JSON.parse(fs.readFileSync(process.argv[2])).valid,cj=JSON.parse(fs.readFileSync(process.argv[3])),wj=JSON.parse(fs.readFileSync(process.argv[4])),vectors=JSON.parse(fs.readFileSync(process.argv[5]))[process.argv[6]],expected=Object.fromEntries(vectors.map(x=>[x.name,x.result])),native=Object.fromEntries(Object.entries(all).filter(([k])=>k in expected)),canonical=Object.fromEntries(Object.entries(cj.valid).map(([k,v])=>[k,v.i64]));if(!u.isDeepStrictEqual(native,expected)||!u.isDeepStrictEqual(p,expected)||!u.isDeepStrictEqual(canonical,expected)||!u.isDeepStrictEqual(wj.valid,expected))process.exit(1);if(!u.isDeepStrictEqual(Object.keys(cj.rejected).sort(),["arity","wrong-type"])||!u.isDeepStrictEqual(wj.malformed,[process.argv[6]+"-empty",process.argv[6]+"-short",process.argv[6]+"-long"]))process.exit(1)' "$work/native.json" "$work/$family-projected.json" "$work/$family-canonical.json" "$work/$family-wasm.json" "$vectors" "$family"
  mkdir "$work/pulp-$family"; cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp-$family/pulp.cell.toml"; cp "$work/$family.wasm" "$work/pulp-$family/pure-function.wasm"
  node "$repo/reference/js/javascript-uab-06-wasm-runner.mjs" "$work/$family.wasm" "$vectors" "$family" --tsv > "$work/$family.tsv"; tab=$(printf '\t')
  : > "$work/$family-pulp-audit.tsv"
  while IFS="$tab" read -r kind name request response; do if [ "$kind" = valid ]; then (cd "$work/pulp-$family" && "$work/pulp-runner" -manifest pulp.cell.toml -provider seme.function-v2 -request "$request") > "$work/$name.log" 2>&1; rg -q "\"response\":\"$response\"" "$work/$name.log"; elif (cd "$work/pulp-$family" && "$work/pulp-runner" -manifest pulp.cell.toml -provider seme.function-v2 -request "$request") >/dev/null 2>&1; then echo "Pulp accepted $name" >&2; exit 1; fi; printf '%s\t%s\n' "$kind" "$name" >> "$work/$family-pulp-audit.tsv"; done < "$work/$family.tsv"
  cmp "$work/$family.tsv" "$work/$family-pulp-audit.tsv" >/dev/null 2>&1 || node -e 'const fs=require("fs"),names=p=>fs.readFileSync(p,"utf8").trim().split("\n").map(x=>x.split("\t").slice(0,2));require("assert").deepStrictEqual(names(process.argv[1]),names(process.argv[2]))' "$work/$family.tsv" "$work/$family-pulp-audit.tsv"
done
sed 's/return (value) =>/return function(value) { return this.base + value; }; \/\//' "$source" > "$work/dynamic-this.js"; if node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/dynamic-this.js" --module "$module" --package bad --revision 1 --entry Immutable --out "$work/bad.g1" 2> "$work/bad.log"; then exit 1; fi; rg -q 'javascript\.' "$work/bad.log"; rg -q 'line|:[1-9][0-9]*:[1-9][0-9]*' "$work/bad.log"
node -e 'const s=require(process.argv[1]),e=["lift","native_parity","target_parity","projection_round_trip","rejection"];require("assert").deepStrictEqual(s.languages.javascript["UAB-06"],e)' "$repo/conformance/uab-v1/scorecard.json"
echo 'JavaScript UAB-06 complete acceptance: all five evidence classes pass'
