#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/javascript-uab11-target.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME

module="$repo/modules/execution/v35/module.g1"
package=seme.uab11/application
language=${UAB11_LANGUAGE:-javascript}
cd "$repo"
if [ "$language" = lua ]; then
  node reference/lua/lua-provider-cli.mjs \
    --source fixtures/lua-uab-11/policy.lua --source fixtures/lua-uab-11/application.lua \
    --module "$module" --package "$package" --revision 1 --entry Apply --out "$work/program.g1"
else
  node reference/js/javascript-package-provider-cli.mjs \
    --files fixtures/javascript-uab-11/application.js,fixtures/javascript-uab-11/policy.js \
    --module "$module" --package "$package" --revision 1 --entry Apply --out "$work/program.g1"
fi
bootstrap/seme-k0-linux-amd64 compiler/g1-compiler.k0 "$work/program.g1" "$work/program.seme"
bootstrap/seme-k0-linux-amd64 compiler/kernel-wire-validator.k0 "$work/program.seme"
if [ "$language" = lua ]; then
  node reference/lua/lua-projector-cli.mjs "$work/program.g1" "$work/projected.lua"
else
  node reference/js/javascript-projector-cli.mjs "$work/program.g1" "$work/projected.mjs"
fi
node reference/js/javascript-uab-11-canonical-corpus.mjs \
  "$repo/fixtures/javascript-uab-11/application.js" "$work/requests.jsonl" "$work/expected.jsonl"
test "$(wc -l < "$work/requests.jsonl" | tr -d ' ')" -eq 2048

(cd reference/go && go build -buildvcs=false -o "$work/codec" ./cmd/pure-application-codec)
(cd reference/go && GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -buildvcs=false -o "$work/canonical-vm.wasm" ./cmd/canonical-wasm-cell)
"$work/codec" encode "$work/program.seme" < "$work/requests.jsonl" > "$work/requests.hex"
test "$(wc -l < "$work/requests.hex" | tr -d ' ')" -eq 2048

# Standalone Wasm receives only the canonical graph and recursive ABI bytes.
# It has no source fixture, source-language runtime, or application-name switch.
node reference/js/canonical-wasm-cell-runner.mjs \
  "$work/canonical-vm.wasm" "$work/program.seme" "$work/requests.hex" > "$work/wasm.hex"
"$work/codec" observe "$work/program.seme" < "$work/wasm.hex" > "$work/wasm.jsonl"
node reference/js/javascript-uab-11-jsonl-equal.mjs "$work/expected.jsonl" "$work/wasm.jsonl"

# Recursive ABI failures reject before evaluation and before any observation.
sed -n '17p' "$work/requests.hex" > "$work/success.hex"
node -e 'const fs=require("fs");const p=process.argv[1];const b=Buffer.from(fs.readFileSync(p,"utf8").trim(),"hex");b[56]=2;fs.writeFileSync(process.argv[2],b.toString("hex")+"\n")' \
  "$work/success.hex" "$work/bad-bool.hex"
node -e 'const fs=require("fs");const b=Buffer.from(fs.readFileSync(process.argv[1],"utf8").trim(),"hex");fs.writeFileSync(process.argv[2],b.subarray(0,b.length-1).toString("hex")+"\n")' \
  "$work/success.hex" "$work/truncated.hex"
node -e 'const fs=require("fs");const b=Buffer.from(fs.readFileSync(process.argv[1],"utf8").trim(),"hex");fs.writeFileSync(process.argv[2],Buffer.concat([b,Buffer.from([0])]).toString("hex")+"\n")' \
  "$work/success.hex" "$work/trailing.hex"
for malformed in bad-bool truncated trailing; do
  if node reference/js/canonical-wasm-cell-runner.mjs "$work/canonical-vm.wasm" \
    "$work/program.seme" "$work/$malformed.hex" > "$work/$malformed.out" 2> "$work/$malformed.log"; then
    echo "standalone Wasm accepted malformed $malformed request" >&2; exit 1
  fi
  test ! -s "$work/$malformed.out"
  rg -q 'canonical_vm.request_invalid' "$work/$malformed.log"
done

# A malformed canonical envelope never establishes an execution boundary.
head -c 31 "$work/program.seme" > "$work/bad-program.seme"
if node reference/js/canonical-wasm-cell-runner.mjs "$work/canonical-vm.wasm" \
  "$work/bad-program.seme" "$work/success.hex" > "$work/bad-program.out" 2> "$work/bad-program.log"; then
  echo 'standalone Wasm accepted malformed canonical graph' >&2; exit 1
fi
test ! -s "$work/bad-program.out"
rg -q 'canonical_vm.graph_invalid' "$work/bad-program.log"

# Capability denial returns no response and no partial trace.
if node reference/js/canonical-wasm-cell-runner.mjs "$work/canonical-vm.wasm" \
  "$work/program.seme" "$work/success.hex" --deny > "$work/denied.out" 2> "$work/denied.log"; then
  echo 'standalone Wasm accepted denied observation' >&2; exit 1
fi
test ! -s "$work/denied.out"
rg -q 'canonical_vm.effect_denied' "$work/denied.log"

# Execute the same cell, graph, ABI corpus, and capability through a pinned
# public Pulp cell host. The proof runner is copied into the pinned checkout so
# its internal imports are governed by that exact Pulp revision.
test -f "$pulp_repo/go.mod"
git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pinned" "$work/pulp"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pinned"
mkdir -p "$work/pinned/cmd/pulp-seme-canonical-vm-proof"
cp targets/wasm/pulp-canonical-vm-v1/runner.go "$work/pinned/cmd/pulp-seme-canonical-vm-proof/main.go"
(cd "$work/pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-canonical-vm-proof)
cp targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml "$work/pulp/pulp.cell.toml"
cp targets/wasm/pulp-canonical-vm-v1/pulp.denied.cell.toml "$work/pulp/pulp.denied.cell.toml"
cp "$work/canonical-vm.wasm" "$work/pulp/canonical-vm.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -graph "$work/program.seme" \
  -requests "$work/requests.hex" > "$work/pulp.raw.jsonl"
node reference/js/pulp-canonical-vm-output.mjs "$work/pulp.raw.jsonl" > "$work/pulp.hex"
"$work/codec" observe "$work/program.seme" < "$work/pulp.hex" > "$work/pulp.jsonl"
node reference/js/javascript-uab-11-jsonl-equal.mjs "$work/expected.jsonl" "$work/pulp.jsonl"

for malformed in bad-bool truncated trailing; do
  if "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -graph "$work/program.seme" \
    -requests "$work/$malformed.hex" > "$work/pulp-$malformed.out" 2> "$work/pulp-$malformed.log"; then
    echo "pinned Pulp accepted malformed $malformed request" >&2; exit 1
  fi
  test ! -s "$work/pulp-$malformed.out"
  rg -q 'canonical_vm.request_invalid' "$work/pulp-$malformed.log"
done
if "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -graph "$work/bad-program.seme" \
  -requests "$work/success.hex" > "$work/pulp-bad-program.out" 2> "$work/pulp-bad-program.log"; then
  echo 'pinned Pulp accepted malformed canonical graph' >&2; exit 1
fi
test ! -s "$work/pulp-bad-program.out"
rg -q 'canonical_vm.graph_invalid' "$work/pulp-bad-program.log"

if "$work/pulp-runner" -manifest "$work/pulp/pulp.denied.cell.toml" -graph "$work/program.seme" \
  -requests "$work/success.hex" > "$work/pulp-denied.out" 2> "$work/pulp-denied.log"; then
  echo 'pinned Pulp accepted denied observation' >&2; exit 1
fi
test ! -s "$work/pulp-denied.out"
rg -q 'canonical_vm.effect_denied' "$work/pulp-denied.log"

echo "$language UAB-11 target: 2048 canonical cases agree through standalone Wasm and pinned Pulp; denial and malformed graph/ABI reject atomically"
