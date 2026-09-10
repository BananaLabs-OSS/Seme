#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb05.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
proxy="$repo/fixtures/go-upb03-offline-proxy"
module=example.test/go-uab-11
root_package="$module/service"
dependency=example.test/seme/checksum
version=v1.2.3
selection="$repo/fixtures/go-upb05-configuration-overlay/configuration-selection.json"
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001

"$repo/scripts/materialize-go-upb05-fixture.sh" "$work/source"
(cd "$work/source" && GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./...)
(cd "$work/source" && GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./service -run '^TestNativeConfigurationCorpus$' -args -native-corpus-dir "$work/corpus")

(cd "$repo/reference/go" &&
  go test -count=1 -buildvcs=false ./goprovider ./goprojector ./canonicaleval ./wasmtarget ./goupb05pipeline ./projectv8instance ./configurationinstance ./configurationexecutor ./cmd/go-upb05-build ./cmd/go-upb05-project ./cmd/go-upb05-configure &&
  go build -buildvcs=false -o "$work/build" ./cmd/go-upb05-build &&
  go build -buildvcs=false -o "$work/project" ./cmd/go-upb05-project &&
  go build -buildvcs=false -o "$work/configure" ./cmd/go-upb05-configure &&
  go build -buildvcs=false -o "$work/observe" ./cmd/canonical-observe &&
  go build -buildvcs=false -o "$work/codec" ./cmd/pure-application-codec &&
  GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -buildvcs=false -o "$work/canonical-vm.wasm" ./cmd/canonical-wasm-cell)

build() {
  source=$1 revision=$2 destination=$3
  GOCACHE="$destination.go-cache" "$work/build" -project "$source" -proxy "$proxy" -module "$module" -package "$root_package" -entry ApplyConfigured -revision "$revision" -out "$destination" \
    -dependency "$dependency" -version "$version" -local-from "$module/application" -local-to "$module/model" -selection "$selection" \
    -execution-g1 "$repo/modules/execution/v36/module.g1" -execution-contract "$repo/modules/execution/v36/module.seme" -foundation-contract "$repo/modules/foundation/v1/module.seme" \
    -package-v4 "$repo/modules/package/v4/module.seme" -project-v8 "$repo/modules/project/v8/module.seme" -dependency-v1 "$repo/modules/dependency/v1/module.seme" -configuration-v3 "$repo/modules/configuration/v3/module.seme" \
    -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0"
}

configure() {
  bundle=$1
  "$work/configure" -bundle "$bundle" -selection "$selection" -foundation "$repo/modules/foundation/v1/module.seme" -execution "$repo/modules/execution/v36/module.seme" -package "$repo/modules/package/v4/module.seme" -dependency "$repo/modules/dependency/v1/module.seme" -configuration "$repo/modules/configuration/v3/module.seme" -project "$repo/modules/project/v8/module.seme" -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0"
}

project() {
  root=$1 bundle=$2 destination=$3
  "$work/project" -root "$root" -to "$destination" -module "$module" -selection "$selection" -manifest "$bundle/COMPLETE.sha256" \
    -construction-v36 "$bundle/construction-v36.g1" -execution-v36 "$bundle/execution-v36.seme" -project-base-v8 "$bundle/project-base-v8.seme" -inventory-v8 "$bundle/inventory-v8.seme" -package-detail-v4 "$bundle/package-detail-v4.seme" -package-v4 "$bundle/package-v4.seme" -dependency-v1 "$bundle/dependency-v1.seme" -configuration-v3 "$bundle/configuration-v3.seme" -project-v8 "$bundle/project-v8.seme" \
    -foundation-contract "$repo/modules/foundation/v1/module.seme" -execution-contract "$repo/modules/execution/v36/module.seme" -package-v4-contract "$repo/modules/package/v4/module.seme" -dependency-contract "$repo/modules/dependency/v1/module.seme" -configuration-v3-contract "$repo/modules/configuration/v3/module.seme" -project-v8-contract "$repo/modules/project/v8/module.seme" -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0"
}

# One v36 run binds source, execution, package ownership, dependency closure,
# configuration, and project authority. Identical offline builds reproduce.
build "$work/source" 1 "$work/bundle-a"
build "$work/source" 1 "$work/bundle-b"
for artifact in execution-v36.seme project-base-v8.seme inventory-v8.seme package-detail-v4.seme package-v4.seme dependency-v1.seme configuration-v3.seme project-v8.seme; do
  cmp "$work/bundle-a/$artifact" "$work/bundle-b/$artifact"
done

# The authenticated configuration plan executes its three initializers in
# dependency order and produces the ready stage-3 service runtime.
configure "$work/bundle-a" < "$repo/fixtures/go-upb05-configuration-overlay/configuration-runtime.json" > "$work/configuration.json"
configure "$work/bundle-a" < "$repo/fixtures/go-upb05-configuration-overlay/configuration-runtime.json" > "$work/configuration-repeat.json"
cmp "$work/configuration.json" "$work/configuration-repeat.json"
node -e '
const fs=require("fs"),r=JSON.parse(fs.readFileSync(process.argv[1])),x=r.execution,a=r.authority;
if(a.contract_revision!=="0000000000000000000000000000e00a"||a.snapshot_revision.length!==64||a.package_count!==5||a.initializer_count!==3)throw Error("project authority");
if(x.outputs.length!==3||x.lifecycle.length!==6)throw Error("initializer count");
if(x.outputs.some((x,i)=>x.order!==i))throw Error("initializer order");
for(let i=0;i<3;i++){const a=x.lifecycle[i*2],b=x.lifecycle[i*2+1];if(a.sequence!==i*2||a.from!=="validated"||a.to!=="initializing"||b.sequence!==i*2+1||b.from!=="initializing"||b.to!=="initialized")throw Error("lifecycle order")}
const runtime=x.outputs[2].result;
if(runtime.kind!=="result"||runtime.variant!=="ok"||runtime.payload.kind!=="record"||runtime.payload.fields.Ready.bool!==true||runtime.payload.fields.Stage.i64!=="3")throw Error("runtime readiness");
const fields=runtime.payload.fields;
if(fields.Settings.fields.NamePrefix.text!=="unit-"||fields.Settings.fields.Limit.i64!=="64"||fields.Policy.fields.Limit.i64!=="64"||fields.Policy.fields.Stage.i64!=="2")throw Error("native configuration parity");
if(fields.State.fields.Name.text!=="configured"||fields.State.fields.Values.items.map(v=>v.i64).join(",")!=="2,3"||fields.State.fields.Counters.entries.length!==1||fields.State.fields.Counters.entries[0].key.i64!=="7"||fields.State.fields.Counters.entries[0].value.i64!=="10")throw Error("runtime input parity");
' "$work/configuration.json"
if printf '%s\n' '{"configuration_inputs":{},"runtime_inputs":{},"capability_grants":[]}' | configure "$work/bundle-a" > "$work/missing-runtime.out" 2> "$work/missing-runtime.err"; then echo 'UPB-05 accepted missing runtime input' >&2; exit 1; fi
test ! -s "$work/missing-runtime.out"
if printf '%s\n' '{"configuration_inputs":{},"runtime_inputs":{"application-state":{"kind":"bool","bool":true}},"capability_grants":[]}' | configure "$work/bundle-a" > "$work/wrong-runtime.out" 2> "$work/wrong-runtime.err"; then echo 'UPB-05 accepted wrong runtime type' >&2; exit 1; fi
test ! -s "$work/wrong-runtime.out"
cp -R "$work/bundle-a" "$work/tampered-configuration"
printf '\000' >> "$work/tampered-configuration/configuration-v3.seme"
if configure "$work/tampered-configuration" < "$repo/fixtures/go-upb05-configuration-overlay/configuration-runtime.json" > "$work/tampered-configuration.out" 2> "$work/tampered-configuration.err"; then echo 'UPB-05 accepted tampered configuration plan' >&2; exit 1; fi
test ! -s "$work/tampered-configuration.out"

# Native, canonical, standalone Wasm, and pinned Pulp agree on all requests.
test "$(wc -l < "$work/corpus/requests.jsonl" | tr -d ' ')" -eq 2058
test "$(wc -l < "$work/corpus/expected.jsonl" | tr -d ' ')" -eq 2058
"$work/observe" "$work/bundle-a/execution-v36.seme" < "$work/corpus/requests.jsonl" > "$work/canonical.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/corpus/expected.jsonl" "$work/canonical.jsonl"
"$work/codec" encode "$work/bundle-a/execution-v36.seme" < "$work/corpus/requests.jsonl" > "$work/requests.hex"
node "$repo/reference/js/canonical-wasm-cell-runner.mjs" "$work/canonical-vm.wasm" "$work/bundle-a/execution-v36.seme" "$work/requests.hex" > "$work/wasm.hex"
"$work/codec" observe "$work/bundle-a/execution-v36.seme" < "$work/wasm.hex" > "$work/wasm.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/corpus/expected.jsonl" "$work/wasm.jsonl"
test -f "$pulp_repo/go.mod"; git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pinned" "$work/pulp"; git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pinned"
mkdir -p "$work/pinned/cmd/pulp-seme-canonical-vm-proof"; cp "$repo/targets/wasm/pulp-canonical-vm-v1/runner.go" "$work/pinned/cmd/pulp-seme-canonical-vm-proof/main.go"
(cd "$work/pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-canonical-vm-proof)
cp "$repo/targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"; cp "$work/canonical-vm.wasm" "$work/pulp/canonical-vm.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -graph "$work/bundle-a/execution-v36.seme" -requests "$work/requests.hex" > "$work/pulp.raw.jsonl"
node "$repo/reference/js/pulp-canonical-vm-output.mjs" "$work/pulp.raw.jsonl" > "$work/pulp.hex"
"$work/codec" observe "$work/bundle-a/execution-v36.seme" < "$work/pulp.hex" > "$work/pulp.jsonl"
node "$repo/reference/js/javascript-uab-11-jsonl-equal.mjs" "$work/corpus/expected.jsonl" "$work/pulp.jsonl"

# Authenticated projection is ordinary Go with opaque module/test bytes intact;
# its semantic execution graph re-lifts exactly.
project "$work/source" "$work/bundle-a" "$work/projected"
cmp "$work/source/go.mod" "$work/projected/go.mod"; cmp "$work/source/go.sum" "$work/projected/go.sum"
cmp "$work/source/application/application_test.go" "$work/projected/application/application_test.go"
cmp "$work/source/service/service_test.go" "$work/projected/service/service_test.go"
cmp "$work/source/service/native_corpus_test.go" "$work/projected/service/native_corpus_test.go"
(cd "$work/projected" && GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./...)
build "$work/projected" 1 "$work/relift"
cmp "$work/bundle-a/execution-v36.seme" "$work/relift/execution-v36.seme"
build "$work/projected" 1 "$work/relift-repeat"
for artifact in execution-v36.seme project-base-v8.seme inventory-v8.seme package-detail-v4.seme package-v4.seme dependency-v1.seme configuration-v3.seme project-v8.seme; do
  cmp "$work/relift/$artifact" "$work/relift-repeat/$artifact"
done

# Direct graph, ABI/capability, source, and output adversaries reject atomically.
head -c 31 "$work/bundle-a/execution-v36.seme" > "$work/bad-graph.seme"; sed -n '17p' "$work/requests.hex" > "$work/one.hex"
if node "$repo/reference/js/canonical-wasm-cell-runner.mjs" "$work/canonical-vm.wasm" "$work/bad-graph.seme" "$work/one.hex" > "$work/bad-graph.out" 2> "$work/bad-graph.err"; then echo 'UPB-05 accepted malformed graph' >&2; exit 1; fi
test ! -s "$work/bad-graph.out"
if node "$repo/reference/js/canonical-wasm-cell-runner.mjs" "$work/canonical-vm.wasm" "$work/bundle-a/execution-v36.seme" "$work/one.hex" --deny > "$work/denied.out" 2> "$work/denied.err"; then echo 'UPB-05 accepted denied effect' >&2; exit 1; fi
test ! -s "$work/denied.out"
cp -R "$work/source" "$work/ambient"; cp "$repo/fixtures/go-upb05-configuration-overlay/adversaries/ambient.go.txt" "$work/ambient/configuration/ambient.go"
if build "$work/ambient" 3 "$work/reject-ambient" > "$work/ambient.out" 2> "$work/ambient.err"; then echo 'UPB-05 accepted ambient configuration' >&2; exit 1; fi
test ! -e "$work/reject-ambient"; test ! -s "$work/ambient.out"
if build "$work/source" 4 "$work/bundle-a" > "$work/existing.out" 2> "$work/existing.err"; then echo 'UPB-05 overwrote output' >&2; exit 1; fi
test ! -s "$work/existing.out"
cp "$work/bundle-a/project-v8.seme" "$work/tampered-project-v8.seme"; printf x >> "$work/tampered-project-v8.seme"
cp -R "$work/bundle-a" "$work/tampered-bundle"; cp "$work/tampered-project-v8.seme" "$work/tampered-bundle/project-v8.seme"
if project "$work/source" "$work/tampered-bundle" "$work/reject-project" > "$work/project-tamper.out" 2> "$work/project-tamper.err"; then echo 'UPB-05 accepted project authority tamper' >&2; exit 1; fi
test ! -e "$work/reject-project"; test ! -s "$work/project-tamper.out"
cp -R "$work/source" "$work/other-source"
sed 's/func DefaultLimit() int64 { return 64 }/func DefaultLimit() int64 { return 63 }/' "$work/other-source/configuration/configuration.go" > "$work/other-source/configuration/configuration.go.next"
mv "$work/other-source/configuration/configuration.go.next" "$work/other-source/configuration/configuration.go"
build "$work/other-source" 1 "$work/bundle-other"
cp -R "$work/bundle-a" "$work/mixed-bundle"
cp "$work/bundle-other/configuration-v3.seme" "$work/mixed-bundle/configuration-v3.seme"
node -e '
const fs=require("fs"),crypto=require("crypto"),d=process.argv[1],names=["construction-v36.g1","execution-v36.seme","project-base-v8.seme","inventory-v8.seme","package-detail-v4.seme","package-v4.seme","dependency-v1.seme","configuration-v3.seme","project-v8.seme"];
let out="seme-go-upb05-bundle-v2\n";for(const n of names)out+=n+" "+crypto.createHash("sha256").update(fs.readFileSync(d+"/"+n)).digest("hex")+"\n";fs.writeFileSync(d+"/COMPLETE.sha256",out);
' "$work/mixed-bundle"
if project "$work/source" "$work/mixed-bundle" "$work/reject-mixed" > "$work/mixed.out" 2> "$work/mixed.err"; then echo 'UPB-05 accepted mixed valid bundle components' >&2; exit 1; fi
test ! -e "$work/reject-mixed"; test ! -s "$work/mixed.out"

echo 'Go UPB-05 core evidence passes over 2,058 typed configuration/lifecycle cases'
