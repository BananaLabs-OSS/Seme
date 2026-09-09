#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd); pulp_repo=${PULP_REPO:-"$repo/../Pulp"}; pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-uab05.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
fixture="$repo/fixtures/go-uab-05"; vectors="$fixture/vectors.json"; module="$repo/modules/execution/v27/module.g1"
(cd "$repo/reference/go" && go test ./goprovider ./goprojector ./canonicaleval ./wasmtarget && go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && go build -buildvcs=false -o "$work/projector" ./cmd/go-projector && go build -buildvcs=false -o "$work/eval" ./cmd/canonical-eval && go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
(cd "$fixture" && go run -buildvcs=false ./cmd/native-observations "$vectors" > "$work/native.json")
"$work/session" --module "$module" --project "$fixture" --package example.test/go-uab-05 --entry Dispatch --revision 1 --out "$work/source.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/source.g1" "$work/program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/kernel-wire-validator.k0" "$work/program.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/program.seme"
"$work/projector" -package dispatch "$work/source.g1" "$work/projected.go"
mkdir -p "$work/projected/cmd/native-observations"; cp "$fixture/go.mod" "$work/projected/go.mod"; cp "$work/projected.go" "$work/projected/program.go"; cp "$fixture/cmd/native-observations/main.go" "$work/projected/cmd/native-observations/main.go"
(cd "$work/projected" && go run -buildvcs=false ./cmd/native-observations "$vectors" > "$work/projected.json")
"$work/session" --module "$module" --project "$work/projected" --package example.test/go-uab-05 --entry Dispatch --revision 1 --out "$work/relift.g1"; cmp "$work/source.g1" "$work/relift.g1"
node "$repo/reference/js/go-uab-05-vectors.mjs" "$vectors" > "$work/canonical-vectors.json"
"$work/eval" "$work/program.seme" "$work/canonical-vectors.json" --named-rejections > "$work/canonical.json"
"$work/lower" "$work/program.seme" "$work/program.wasm" "$work/abi.json"
node "$repo/reference/js/go-uab-05-wasm-runner.mjs" "$work/program.wasm" "$vectors" > "$work/wasm.json"
node "$repo/reference/js/go-uab-05-observation-compare.mjs" "$vectors" "$work/native.json" "$work/projected.json" "$work/canonical.json" "$work/wasm.json"

test -f "$pulp_repo/go.mod"; git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"; mkdir "$work/pulp" "$work/pinned"; git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pinned"
(cd "$work/pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp/pulp.cell.toml"; cp "$work/program.wasm" "$work/pulp/pure-function.wasm"
node "$repo/reference/js/go-uab-05-wasm-runner.mjs" "$work/program.wasm" "$vectors" --tsv > "$work/requests.tsv"; : > "$work/pulp-audit.tsv"; tab=$(printf '\t')
while IFS="$tab" read -r kind name request response; do
  if [ "$kind" = valid ]; then "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 -request "$request" > "$work/pulp-$name.log" 2>&1; rg -q "\"response\":\"$response\"" "$work/pulp-$name.log"; else if "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 -request "$request" > "$work/pulp-$name.log" 2>&1; then echo "Pulp accepted $name" >&2; exit 1; fi; fi
  printf '%s\t%s\n' "$kind" "$name" >> "$work/pulp-audit.tsv"
done < "$work/requests.tsv"
node -e 'const fs=require("fs"),v=require(process.argv[1]),rows=fs.readFileSync(process.argv[2],"utf8").trim().split("\n").map(x=>x.split("\t")),e=[...v.valid.map(x=>["valid",x.name]),...v.malformed.map(x=>["malformed",x.name])];require("assert").deepStrictEqual(rows,e)' "$vectors" "$work/pulp-audit.tsv"

reject() { name=$1; pattern=$2; if "$work/session" --module "$module" --project "$work/$name" --package example.test/go-uab-05 --entry Dispatch --revision 1 --out "$work/$name.g1" 2> "$work/$name.log"; then echo "Go UAB-05 accepted $name" >&2; exit 1; fi; rg -q "$pattern" "$work/$name.log"; rg -q 'File:"program.go", Line:[1-9][0-9]*' "$work/$name.log"; printf 'source-reject\t%s\n' "$name" >> "$work/source-audit.tsv"; }
: > "$work/source-audit.tsv"
cp -R "$fixture" "$work/missing-method"; sed -i '0,/Adjust(value int64)/s//Other(value int64)/' "$work/missing-method/program.go"; reject missing-method 'go.type'
cp -R "$fixture" "$work/signature-mismatch"; sed -i '0,/Adjust(value int64) int64/s//Adjust(value int64) bool/' "$work/signature-mismatch/program.go"; reject signature-mismatch 'go.type'
node -e 'const fs=require("fs"),a=fs.readFileSync(process.argv[1],"utf8").trim().split("\n");require("assert").deepStrictEqual(a,["source-reject\tmissing-method","source-reject\tsignature-mismatch"])' "$work/source-audit.tsv"
node -e 'const s=require(process.argv[1]),e=["lift","native_parity","target_parity","projection_round_trip","rejection"];if(JSON.stringify(s.languages.go["UAB-05"])!==JSON.stringify(e))process.exit(1)' "$repo/conformance/uab-v1/scorecard.json"
echo 'Go UAB-05 complete acceptance: all five evidence classes pass'
