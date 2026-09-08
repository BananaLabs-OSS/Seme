#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v27.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; export XDG_CACHE_HOME

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule ./goprovider ./wasmtarget && \
 go run -buildvcs=false ./cmd/execution-module-v27 > "$work/module.g1" && \
 go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && \
 go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
cmp "$repo/modules/execution/v27/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v27" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v27/module.seme" "$work/module.seme"
(cd "$repo/fixtures/go-execution-v27" && go test -buildvcs=false ./...)
(cd "$repo/reference/js" && npm test)

"$work/session" --module "$repo/modules/execution/v27/module.g1" \
  --project "$repo/fixtures/go-execution-v27" --package example.com/seme-interface-dispatch-proof \
  --entry Dispatch --revision 1 --out "$work/go.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/go.g1" "$work/go.seme"
node "$repo/reference/js/javascript-projector-cli.mjs" "$work/go.g1" "$work/projected.mjs"
node --check "$work/projected.mjs"
node "$repo/reference/js/interface-native-runner.mjs" "$work/projected.mjs"
node "$repo/reference/js/javascript-provider-cli.mjs" --source "$work/projected.mjs" \
  --module "$repo/modules/execution/v27/module.g1" --package example.com/seme-interface-dispatch-proof \
  --entry Dispatch --revision 1 --out "$work/js.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/js.g1" "$work/js.seme"
cmp "$work/go.seme" "$work/js.seme"

"$work/lower" "$work/go.seme" "$work/dispatch.wasm" "$work/dispatch-abi.json"
"$work/lower" "$work/go.seme" "$work/dispatch-second.wasm" "$work/dispatch-second-abi.json"
cmp "$work/dispatch.wasm" "$work/dispatch-second.wasm"
cmp "$work/dispatch-abi.json" "$work/dispatch-second-abi.json"
rg -q '"request_size": 17' "$work/dispatch-abi.json"
rg -q '"response_size": 8' "$work/dispatch-abi.json"
node "$repo/reference/js/interface-dispatch-runner.mjs" "$work/dispatch.wasm"

"$work/session" --module "$repo/modules/execution/v27/module.g1" \
  --project "$repo/fixtures/go-execution-v27" --package example.com/seme-interface-dispatch-proof \
  --entry Apply --revision 1 --out "$work/apply.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/apply.g1" "$work/apply.seme"
"$work/lower" "$work/apply.seme" "$work/apply.wasm" "$work/apply-abi.json"
rg -q '"request_size": 24' "$work/apply-abi.json"
rg -q '"response_size": 8' "$work/apply-abi.json"
node "$repo/reference/js/interface-witness-runner.mjs" "$work/apply.wasm"

witness=$(awk '$3 == "0000000000000000000000000000a012" { print $2; exit }' "$work/go.g1")
other_method=$(awk '$3 == "0000000000000000000000000000a002" { print $2 }' "$work/go.g1" | tail -1)
integer=$(awk '$3 == "00000000000000000000000000009010" { print $2; exit }' "$work/go.g1")
boolean=$(awk '$3 == "00000000000000000000000000009020" { print $2; exit }' "$work/go.g1")
requirement=$(awk '$3 == "0000000000000000000000000000a011" { print $2; exit }' "$work/go.g1")

# A witness may not substitute a method belonging to another concrete receiver.
awk -v witness="$witness" -v replacement="$other_method" '
  $1 == "en" { inside = ($2 == witness) }
  inside && $1 == "fi" && $2 == "000000000000000000000000000a0122" { methods = 1 }
  inside && methods && $1 == "rf" && !done { print "rf " replacement; done = 1; next }
  { print }
' "$work/go.g1" > "$work/wrong-witness.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/wrong-witness.g1" "$work/wrong-witness.seme"
if "$work/lower" "$work/wrong-witness.seme" "$work/wrong-witness.wasm" "$work/wrong-witness.json" >/dev/null 2>&1; then
  echo "receiver-incompatible witness was accepted" >&2
  exit 1
fi

# Requirement and implementation signatures must agree exactly.
sed "s/fi 000000000000000000000000000a0112 rf $integer/fi 000000000000000000000000000a0112 rf $boolean/" \
  "$work/go.g1" > "$work/wrong-signature.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/wrong-signature.g1" "$work/wrong-signature.seme"
if "$work/lower" "$work/wrong-signature.seme" "$work/wrong-signature.wasm" "$work/wrong-signature.json" >/dev/null 2>&1; then
  echo "interface signature mismatch was accepted" >&2
  exit 1
fi

# A dynamic call must name a requirement belonging to its interface.
sed "s/fi 000000000000000000000000000a0141 rf $requirement/fi 000000000000000000000000000a0141 rf $other_method/" \
  "$work/go.g1" > "$work/wrong-requirement.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/wrong-requirement.g1" "$work/wrong-requirement.seme"
if "$work/lower" "$work/wrong-requirement.seme" "$work/wrong-requirement.wasm" "$work/wrong-requirement.json" >/dev/null 2>&1; then
  echo "non-requirement dynamic call target was accepted" >&2
  exit 1
fi

mkdir "$work/pulp" "$work/pulp-pinned"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pulp-pinned"
(cd "$work/pulp-pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof)
cp "$repo/targets/wasm/pulp-function-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/dispatch.wasm" "$work/pulp/pure-function.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 \
  -request 0003000000000000000400000000000000 > "$work/pulp-offset.log" 2>&1
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v1 \
  -request 0103000000000000000400000000000000 > "$work/pulp-scale.log" 2>&1
rg -q '"response":"0700000000000000"' "$work/pulp-offset.log"
rg -q '"response":"0c00000000000000"' "$work/pulp-scale.log"

version=2
while [ "$version" -le 26 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo "Core Execution v27: native Go and JavaScript interfaces shared witness-certified dispatch through Wasm and Pulp"
