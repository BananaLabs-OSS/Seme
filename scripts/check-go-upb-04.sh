#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb04.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
proxy="$repo/fixtures/go-upb03-offline-proxy"
module=example.test/go-uab-11
root_package="$module/application"
dependency=example.test/seme/checksum
version=v1.2.3
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001

"$repo/scripts/materialize-go-upb04-fixture.sh" "$work/source"
(cd "$work/source" && GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./...)

(cd "$repo/reference/go" &&
  go test -count=1 -buildvcs=false ./goupb04pipeline ./gopackagev3adapter ./packagecallinstance ./packagev3instance ./projectv5instance ./projectv5report ./goprojector ./cmd/go-upb04-build ./cmd/go-upb04-project ./cmd/project-v5-report &&
  go build -buildvcs=false -o "$work/build" ./cmd/go-upb04-build &&
  go build -buildvcs=false -o "$work/project" ./cmd/go-upb04-project &&
  go build -buildvcs=false -o "$work/report" ./cmd/project-v5-report &&
  go build -buildvcs=false -o "$work/observe" ./cmd/canonical-observe &&
  go build -buildvcs=false -o "$work/codec" ./cmd/pure-application-codec &&
  GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -buildvcs=false -o "$work/canonical-vm.wasm" ./cmd/canonical-wasm-cell)

build() {
  source=$1 revision=$2 destination=$3
  GOCACHE="$destination.go-cache" "$work/build" -project "$source" -proxy "$proxy" -module "$module" -package "$root_package" -entry Apply -revision "$revision" -out "$destination" \
    -dependency "$dependency" -version "$version" -local-from "$root_package" -local-to "$module/model" \
    -execution-g1 "$repo/modules/execution/v35/module.g1" -execution-contract "$repo/modules/execution/v35/module.seme" \
    -package-v1 "$repo/modules/package/v1/module.seme" -package-v2 "$repo/modules/package/v2/module.seme" -package-v3 "$repo/modules/package/v3/module.seme" \
    -project-v1 "$repo/modules/project/v1/module.seme" -project-v2 "$repo/modules/project/v2/module.seme" -project-v3 "$repo/modules/project/v3/module.seme" -project-v4 "$repo/modules/project/v4/module.seme" -project-v5 "$repo/modules/project/v5/module.seme" \
    -dependency-v1 "$repo/modules/dependency/v1/module.seme" -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0"
}
report() {
  b=$1 out=$2
  "$work/report" --execution "$repo/modules/execution/v35/module.seme" --package-v1 "$repo/modules/package/v1/module.seme" --package-v2 "$repo/modules/package/v2/module.seme" --package-v3-contract "$repo/modules/package/v3/module.seme" \
    --dependency-contract "$repo/modules/dependency/v1/module.seme" --project-v2-contract "$repo/modules/project/v2/module.seme" --project-v3-contract "$repo/modules/project/v3/module.seme" --project-v4-contract "$repo/modules/project/v4/module.seme" --project-v5-contract "$repo/modules/project/v5/module.seme" \
    --project "$b/project-v1.seme" --inventory "$b/inventory-v2.seme" --package-graph "$b/package-v2.seme" --project-v3 "$b/project-v3.seme" --dependency "$b/dependency-v1.seme" --project-v4 "$b/project-v4.seme" --package-v3 "$b/package-v3.seme" --composed "$b/project-v5.seme" > "$out"
}
project() {
  root=$1 bundle=$2 destination=$3
  "$work/project" -root "$root" -construction "$bundle/construction.g1" -package-v2 "$bundle/package-v2.seme" -package-v3 "$bundle/package-v3.seme" -module "$module" -to "$destination" \
    -execution-contract "$repo/modules/execution/v35/module.seme" -package-v3-contract "$repo/modules/package/v3/module.seme" -dependency-contract "$repo/modules/dependency/v1/module.seme" -project-v5-contract "$repo/modules/project/v5/module.seme"
}

# Lift and resolution are deterministic; client revisions are not semantic.
build "$work/source" 1 "$work/bundle-a"
build "$work/source" 99 "$work/bundle-b"
for artifact in project-v1.seme package-v2.seme dependency-v1.seme package-v3.seme project-v5.seme; do
  cmp "$work/bundle-a/$artifact" "$work/bundle-b/$artifact"
done
report "$work/bundle-a" "$work/report-a.json"
report "$work/bundle-a" "$work/report-b.json"
cmp "$work/report-a.json" "$work/report-b.json"
node -e '
const fs=require("fs"),r=JSON.parse(fs.readFileSync(process.argv[1]));
if(r.contract_revision!=="0000000000000000000000000000e005")throw Error("project-v5 revision");
const d=r.complete_package_graph.declarations;
const owners=new Set(d.map(x=>x.owner));
for(const p of ["example.test/go-uab-11/application","example.test/go-uab-11/model","example.test/go-uab-11/policy"])if(!owners.has(p))throw Error("missing owner "+p);
for(const k of ["data-type","behavioral-interface","receiver-callable","generic-realization"])if(!d.some(x=>x.kind===k))throw Error("missing kind "+k);
if(d.some(x=>x.owner.endsWith("/model")&&Array.isArray(x.imports)&&x.imports.some(i=>i.resolved.endsWith("/application"))))throw Error("generic argument became reverse import");
' "$work/report-a.json"

# The complete 2,048-command corpus agrees canonically and through both target paths.
node "$repo/reference/js/javascript-uab-11-canonical-corpus.mjs" "$repo/fixtures/javascript-uab-11/application.js" "$work/requests.jsonl" "$work/expected.jsonl"
test "$(wc -l < "$work/requests.jsonl" | tr -d ' ')" -eq 2048
"$work/observe" "$work/bundle-a/project-v1.seme" < "$work/requests.jsonl" > "$work/canonical.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/expected.jsonl" "$work/canonical.jsonl"
"$work/codec" encode "$work/bundle-a/project-v1.seme" < "$work/requests.jsonl" > "$work/requests.hex"
node "$repo/reference/js/canonical-wasm-cell-runner.mjs" "$work/canonical-vm.wasm" "$work/bundle-a/project-v1.seme" "$work/requests.hex" > "$work/wasm.hex"
"$work/codec" observe "$work/bundle-a/project-v1.seme" < "$work/wasm.hex" > "$work/wasm.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/expected.jsonl" "$work/wasm.jsonl"
test -f "$pulp_repo/go.mod"; git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pinned" "$work/pulp"; git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pinned"
mkdir -p "$work/pinned/cmd/pulp-seme-canonical-vm-proof"; cp "$repo/targets/wasm/pulp-canonical-vm-v1/runner.go" "$work/pinned/cmd/pulp-seme-canonical-vm-proof/main.go"
(cd "$work/pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-canonical-vm-proof)
cp "$repo/targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"; cp "$work/canonical-vm.wasm" "$work/pulp/canonical-vm.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -graph "$work/bundle-a/project-v1.seme" -requests "$work/requests.hex" > "$work/pulp.raw.jsonl"
node "$repo/reference/js/pulp-canonical-vm-output.mjs" "$work/pulp.raw.jsonl" > "$work/pulp.hex"
"$work/codec" observe "$work/bundle-a/project-v1.seme" < "$work/pulp.hex" > "$work/pulp.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/expected.jsonl" "$work/pulp.jsonl"

# Authenticated projection remains an ordinary three-package Go project and re-lifts exactly.
project "$work/source" "$work/bundle-a" "$work/projected"
cmp "$work/source/go.mod" "$work/projected/go.mod"; cmp "$work/source/go.sum" "$work/projected/go.sum"
cmp "$work/source/application/application_test.go" "$work/projected/application/application_test.go"
(cd "$work/projected" && GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./...)
build "$work/projected" 7 "$work/relift-a"; build "$work/projected" 88 "$work/relift-b"
cmp "$work/bundle-a/project-v1.seme" "$work/relift-a/project-v1.seme"
cmp "$work/relift-a/package-v3.seme" "$work/relift-b/package-v3.seme"
cmp "$work/relift-a/project-v5.seme" "$work/relift-b/project-v5.seme"

# Direct artifact, filesystem, dependency, output, graph, ABI, and capability adversaries fail atomically.
reject_build() { src=$1 dest=$2; if build "$src" 3 "$dest" >"$dest.out" 2>"$dest.err"; then echo "UPB-04 accepted invalid source" >&2; exit 1; fi; test ! -e "$dest"; test ! -s "$dest.out"; }
cp -R "$work/source" "$work/badsum"; sed 's/h1:[A-Za-z0-9+\/=]*/h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA/' "$work/badsum/go.sum" > "$work/badsum/go.sum.x"; mv "$work/badsum/go.sum.x" "$work/badsum/go.sum"; reject_build "$work/badsum" "$work/reject-badsum"
cp -R "$work/source" "$work/symlink"; mv "$work/symlink/model/model.go" "$work/model.go"; ln -s "$work/model.go" "$work/symlink/model/model.go"; reject_build "$work/symlink" "$work/reject-symlink"
cp "$work/bundle-a/package-v3.seme" "$work/tampered-package-v3.seme"; printf x >> "$work/tampered-package-v3.seme"
if "$work/project" -root "$work/source" -construction "$work/bundle-a/construction.g1" -package-v2 "$work/bundle-a/package-v2.seme" -package-v3 "$work/tampered-package-v3.seme" -module "$module" -to "$work/reject-project" -execution-contract "$repo/modules/execution/v35/module.seme" -package-v3-contract "$repo/modules/package/v3/module.seme" -dependency-contract "$repo/modules/dependency/v1/module.seme" -project-v5-contract "$repo/modules/project/v5/module.seme" > "$work/project-tamper.out" 2> "$work/project-tamper.err"; then echo 'UPB-04 accepted ownership tamper' >&2; exit 1; fi
test ! -e "$work/reject-project"; test ! -s "$work/project-tamper.out"
if build "$work/source" 4 "$work/bundle-a" > "$work/existing.out" 2> "$work/existing.err"; then echo 'UPB-04 overwrote output' >&2; exit 1; fi
head -c 31 "$work/bundle-a/project-v1.seme" > "$work/bad-graph.seme"; sed -n '17p' "$work/requests.hex" > "$work/one.hex"
if node "$repo/reference/js/canonical-wasm-cell-runner.mjs" "$work/canonical-vm.wasm" "$work/bad-graph.seme" "$work/one.hex" > "$work/bad-graph.out" 2> "$work/bad-graph.err"; then echo 'UPB-04 accepted malformed graph' >&2; exit 1; fi
test ! -s "$work/bad-graph.out"
if node "$repo/reference/js/canonical-wasm-cell-runner.mjs" "$work/canonical-vm.wasm" "$work/bundle-a/project-v1.seme" "$work/one.hex" --deny > "$work/denied.out" 2> "$work/denied.err"; then echo 'UPB-04 accepted denied effect' >&2; exit 1; fi
test ! -s "$work/denied.out"

echo 'Go UPB-04: seven independent evidence classes pass over 2,048 typed cross-package cases'
