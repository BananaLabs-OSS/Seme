#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd); pulp_repo=${PULP_REPO:-"$repo/../Pulp"}; pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-uab06.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
fixture="$repo/fixtures/go-uab-06"; vectors="$fixture/vectors.json"; module="$repo/modules/execution/v29/module.g1"
(cd "$repo/reference/go" && go test -buildvcs=false ./goprojector ./canonicaleval ./wasmtarget && go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && go build -buildvcs=false -o "$work/projector" ./cmd/go-projector && go build -buildvcs=false -o "$work/eval" ./cmd/canonical-eval && go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
(cd "$fixture" && go run -buildvcs=false ./cmd/native-observations "$vectors" > "$work/native.json")
test -f "$pulp_repo/go.mod"; git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"; mkdir "$work/pulp-pinned"; git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"; (cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
for pair in ImmutableRun:immutable MutableRun:mutable; do
  entry=${pair%%:*}; family=${pair#*:}
  "$work/session" --module "$module" --project "$fixture" --package example.test/go-uab-06 --entry "$entry" --revision 1 --out "$work/$family.g1"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/$family.g1" "$work/$family.seme"
  "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/$family.seme"
  "$work/projector" -package closures "$work/$family.g1" "$work/$family.go"
  mkdir -p "$work/projected-$family/cmd/native-observations"; cp "$fixture/go.mod" "$work/projected-$family/go.mod"; cp "$work/$family.go" "$work/projected-$family/program.go"; cp "$fixture/cmd/native-observations/main.go" "$work/projected-$family/cmd/native-observations/main.go"
  (cd "$work/projected-$family" && go run -buildvcs=false ./cmd/native-observations "$vectors" > "$work/$family-projected.json")
  "$work/session" --module "$module" --project "$work/projected-$family" --package example.test/go-uab-06 --entry "$entry" --revision 1 --out "$work/$family-relift.g1"; cmp "$work/$family.g1" "$work/$family-relift.g1"
  node "$repo/reference/js/go-uab-06-adapter.mjs" canonical "$vectors" "$family" > "$work/$family-vectors.json"; "$work/eval" "$work/$family.seme" "$work/$family-vectors.json" --named-rejections > "$work/$family-canonical.json"
  "$work/lower" "$work/$family.seme" "$work/$family.wasm" "$work/$family-abi.json"; node "$repo/reference/js/go-uab-06-wasm.mjs" "$work/$family.wasm" "$vectors" "$family" > "$work/$family-wasm.json"
  node -e 'const fs=require("fs"),u=require("util"),family=process.argv[6],native=JSON.parse(fs.readFileSync(process.argv[1]))[family],projected=JSON.parse(fs.readFileSync(process.argv[2]))[family],canonical=JSON.parse(fs.readFileSync(process.argv[3])),wasm=JSON.parse(fs.readFileSync(process.argv[4])),vectors=JSON.parse(fs.readFileSync(process.argv[5])),expected=Object.fromEntries(vectors[family].map(x=>[x.name,x.result])),c=Object.fromEntries(Object.entries(canonical.valid).map(([k,v])=>[k,v.i64])),bad=vectors.malformed.map(x=>x.name).sort();for(const got of [native.valid,projected.valid,c,wasm.valid])if(!u.isDeepStrictEqual(got,expected))process.exit(1);for(const got of [Object.keys(native.rejected).sort(),Object.keys(projected.rejected).sort(),Object.keys(canonical.rejected).sort(),Object.keys(wasm.rejected).sort()])if(!u.isDeepStrictEqual(got,bad))process.exit(1)' "$work/native.json" "$work/$family-projected.json" "$work/$family-canonical.json" "$work/$family-wasm.json" "$vectors" "$family"
  mkdir "$work/pulp-$family"; cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp-$family/pulp.cell.toml"; cp "$work/$family.wasm" "$work/pulp-$family/pure-function.wasm"
  node "$repo/reference/js/go-uab-06-wasm.mjs" "$work/$family.wasm" "$vectors" "$family" --tsv > "$work/$family.tsv"; : > "$work/$family-pulp.tsv"; tab=$(printf '\t')
  while IFS="$tab" read -r kind name request response; do if [ "$kind" = valid ]; then (cd "$work/pulp-$family" && "$work/pulp-runner" -manifest pulp.cell.toml -provider seme.function-v2 -request "$request") > "$work/$family-$name.log" 2>&1; rg -q "\"response\":\"$response\"" "$work/$family-$name.log"; elif (cd "$work/pulp-$family" && "$work/pulp-runner" -manifest pulp.cell.toml -provider seme.function-v2 -request "$request") >/dev/null 2>&1; then echo "Pulp accepted $family $name" >&2; exit 1; fi; printf '%s\t%s\n' "$kind" "$name" >> "$work/$family-pulp.tsv"; done < "$work/$family.tsv"
  node -e 'const fs=require("fs"),names=p=>fs.readFileSync(p,"utf8").trim().split("\n").map(x=>x.split("\t").slice(0,2));require("assert").deepStrictEqual(names(process.argv[1]),names(process.argv[2]))' "$work/$family.tsv" "$work/$family-pulp.tsv"
done
cp -R "$fixture" "$work/unsupported"; sed -i 's/value = value + delta; return value/value = value + delta; go func() {}(); return value/' "$work/unsupported/program.go"
if "$work/session" --module "$module" --project "$work/unsupported" --package example.test/go-uab-06 --entry MutableRun --revision 1 --out "$work/bad.g1" 2> "$work/bad.log"; then echo 'Go UAB-06 accepted goroutine capture' >&2; exit 1; fi
rg -q 'go\.|expression\.' "$work/bad.log"; rg -q 'File:"program.go", Line:[1-9][0-9]*' "$work/bad.log"
node -e 'const s=require(process.argv[1]),e=["lift","native_parity","target_parity","projection_round_trip","rejection"];require("assert").deepStrictEqual(s.languages.go["UAB-06"],e)' "$repo/conformance/uab-v1/scorecard.json"
echo 'Go UAB-06 complete acceptance: all five evidence classes pass'
