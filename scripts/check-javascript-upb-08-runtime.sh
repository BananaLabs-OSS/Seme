#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd); fixture="$repo/fixtures/javascript-upb05-configuration"; vectors="$fixture/transport-vectors.json"
pulp_repo=${PULP_REPO:-"$repo/../Pulp"};pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-upb08-runtime.XXXXXX");trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache";XDG_CACHE_HOME="$work/cache";export GOCACHE XDG_CACHE_HOME
(cd "$fixture"&&node "$repo/reference/js/javascript-package-provider-cli.mjs" --files application.js,configuration.js,controlled.js,policy.js,state.js,transport.js --module "$repo/modules/execution/v36/module.g1" --package example.test/javascript-upb05 --entry Dispatch --revision 1 --out "$work/program.g1")
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/program.g1" "$work/program.seme"
node "$repo/reference/js/javascript-upb08-native.mjs" "$fixture" "$vectors" > "$work/native.json"
(cd "$repo/reference/go"&&go build -buildvcs=false -o "$work/eval" ./cmd/canonical-eval&&go build -buildvcs=false -o "$work/codec" ./cmd/pure-application-codec&&GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -buildvcs=false -o "$work/vm.wasm" ./cmd/canonical-wasm-cell)
"$work/eval" "$work/program.seme" "$vectors" --named-rejections > "$work/canonical.json"
node -e 'const f=require("fs"),a=JSON.parse(f.readFileSync(process.argv[1])),b=JSON.parse(f.readFileSync(process.argv[2]));require("assert").deepStrictEqual(a,b)' "$work/native.json" "$work/canonical.json"
node - "$vectors" "$work/requests.jsonl" "$work/expected.jsonl" <<'NODE'
const fs=require("fs"),v=JSON.parse(fs.readFileSync(process.argv[2]));fs.writeFileSync(process.argv[3],v.valid.map(x=>JSON.stringify({arguments:x.arguments})).join("\n")+"\n");fs.writeFileSync(process.argv[4],v.valid.map(x=>JSON.stringify({value:x.result,effects:[]})).join("\n")+"\n");
NODE
"$work/codec" encode "$work/program.seme" < "$work/requests.jsonl" > "$work/requests.hex";node "$repo/reference/js/canonical-wasm-cell-runner.mjs" "$work/vm.wasm" "$work/program.seme" "$work/requests.hex" > "$work/wasm.hex";"$work/codec" observe "$work/program.seme" < "$work/wasm.hex" > "$work/wasm.jsonl";cmp "$work/expected.jsonl" "$work/wasm.jsonl"
test -f "$pulp_repo/go.mod";git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}";mkdir "$work/pinned" "$work/pulp";git -C "$pulp_repo" archive "$pulp_commit"|tar -x -C "$work/pinned";mkdir -p "$work/pinned/cmd/pulp-seme-canonical-vm-proof";cp "$repo/targets/wasm/pulp-canonical-vm-v1/runner.go" "$work/pinned/cmd/pulp-seme-canonical-vm-proof/main.go";(cd "$work/pinned"&&go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-canonical-vm-proof);cp "$repo/targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml";cp "$work/vm.wasm" "$work/pulp/canonical-vm.wasm";"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -graph "$work/program.seme" -requests "$work/requests.hex" > "$work/pulp.raw.jsonl";node "$repo/reference/js/pulp-canonical-vm-output.mjs" "$work/pulp.raw.jsonl" > "$work/pulp.hex";"$work/codec" observe "$work/program.seme" < "$work/pulp.hex" > "$work/pulp.jsonl";cmp "$work/expected.jsonl" "$work/pulp.jsonl"
(cd "$repo/reference/go"&&go test -count=1 ./orderedtransportruntime ./orderedtransportplacement)
echo 'JavaScript UPB-08 runtime: native JS, canonical Seme, Wasm, and pinned Pulp dispatch agree; neutral transport host and placement boundaries pass'
