#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb07.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
proxy="$repo/fixtures/go-upb03-offline-proxy"
selection="$repo/fixtures/go-upb05-configuration-overlay/configuration-selection.json"
durable_selection="$repo/fixtures/go-upb07-durable-overlay/durable-selection.json"

# Class 1: the complete prior useful-project bridge remains authoritative.
"$repo/scripts/check-go-upb-06.sh"

# Class 2: fixture semantics and all contract artifacts reproduce independently.
"$repo/scripts/check-go-upb-07-fixture.sh"
"$repo/scripts/check-durable-state-v1.sh"
"$repo/scripts/check-source-presentation-v1.sh"
"$repo/scripts/check-project-contract-v10.sh"

# Class 3: planner target parity plus the separately bounded host-only port
# profile probe. This does not claim that Pulp invokes DurablePort.
"$repo/scripts/check-go-upb-07-runtime.sh"
"$repo/scripts/check-go-upb-07-port-runtime.sh"
"$repo/scripts/check-go-upb-07-pulp-placement.sh"

# Class 4: focused producer, authenticated consumer, projection, and report APIs.
"$repo/scripts/materialize-go-upb07-fixture.sh" "$work/source"
(cd "$repo/reference/go" &&
  go test -p=1 -count=1 ./goupb07pipeline ./goupb07bundle ./godurableadapter ./gopresentationadapter ./durableinstance ./presentationinstance ./projectv10instance ./goprojector ./cmd/go-upb07-build ./cmd/go-upb07-project ./cmd/go-upb07-report &&
  go build -buildvcs=false -o "$work/build" ./cmd/go-upb07-build &&
  go build -buildvcs=false -o "$work/project" ./cmd/go-upb07-project &&
  go build -buildvcs=false -o "$work/report" ./cmd/go-upb07-report)

build() {
  source=$1 revision=$2 destination=$3
  "$work/build" -project "$source" -proxy "$proxy" -module example.test/go-uab-11 -package example.test/go-uab-11/service -entry ApplyConfiguredResource -revision "$revision" -out "$destination" \
    -dependency example.test/seme/checksum -version v1.2.3 -local-from example.test/go-uab-11/application -local-to example.test/go-uab-11/model \
    -selection "$selection" -resources "$source/resources.json" -resource-owner example.test/go-uab-11/service -durable-selection "$source/durable-selection.json" \
    -execution-g1 "$repo/modules/execution/v36/module.g1" -execution-contract "$repo/modules/execution/v36/module.seme" -foundation-contract "$repo/modules/foundation/v1/module.seme" \
    -package-v4 "$repo/modules/package/v4/module.seme" -project-v8 "$repo/modules/project/v8/module.seme" -dependency-v1 "$repo/modules/dependency/v1/module.seme" -configuration-v3 "$repo/modules/configuration/v3/module.seme" \
    -resource-v1 "$repo/modules/resource/v1/module.seme" -project-v9 "$repo/modules/project/v9/module.seme" -durable-state-v1 "$repo/modules/durable-state/v1/module.seme" \
    -source-presentation-v1 "$repo/modules/source-presentation/v1/module.seme" -project-v10 "$repo/modules/project/v10/module.seme" \
    -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0"
}
report() {
  bundle=$1
  "$work/report" -bundle "$bundle" -selection "$selection" -durable-selection "$durable_selection" -foundation "$repo/modules/foundation/v1/module.seme" -execution "$repo/modules/execution/v36/module.seme" \
    -package "$repo/modules/package/v4/module.seme" -dependency "$repo/modules/dependency/v1/module.seme" -configuration "$repo/modules/configuration/v3/module.seme" \
    -resource "$repo/modules/resource/v1/module.seme" -durable-state "$repo/modules/durable-state/v1/module.seme" -presentation "$repo/modules/source-presentation/v1/module.seme" \
    -project-v8 "$repo/modules/project/v8/module.seme" -project-v9 "$repo/modules/project/v9/module.seme" -project-v10 "$repo/modules/project/v10/module.seme" \
    -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0"
}
project() {
  root=$1 bundle=$2 destination=$3
  "$work/project" -root "$root" -to "$destination" -module example.test/go-uab-11 -bundle "$bundle" \
    -selection "$selection" -durable-selection "$durable_selection" -foundation "$repo/modules/foundation/v1/module.seme" -execution "$repo/modules/execution/v36/module.seme" \
    -package "$repo/modules/package/v4/module.seme" -dependency "$repo/modules/dependency/v1/module.seme" -configuration "$repo/modules/configuration/v3/module.seme" \
    -resource "$repo/modules/resource/v1/module.seme" -durable-state "$repo/modules/durable-state/v1/module.seme" -source-presentation "$repo/modules/source-presentation/v1/module.seme" \
    -project-v8 "$repo/modules/project/v8/module.seme" -project-v9 "$repo/modules/project/v9/module.seme" -project-v10 "$repo/modules/project/v10/module.seme" \
    -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0"
}

# Class 5: identical offline inputs reproduce the complete fourteen-artifact bundle.
build "$work/source" 1 "$work/bundle-a"
build "$work/source" 1 "$work/bundle-b"
diff -ru "$work/bundle-a" "$work/bundle-b"

# Class 6: source-free reporting is deterministic and closed-world tampering fails silently.
report "$work/bundle-a" > "$work/report-a.json"
report "$work/bundle-a" > "$work/report-b.json"
cmp "$work/report-a.json" "$work/report-b.json"
node - "$work/report-a.json" <<'NODE'
const fs=require("fs"), r=JSON.parse(fs.readFileSync(process.argv[2]));
const fail=x=>{throw Error("UPB07 authority: "+x)}, hex32=x=>typeof x==="string"&&/^[0-9a-f]{32}$/.test(x);
if(r.project_contract_revision!=="0000000000000000000000000000e00e"||r.durable_contract_revision!=="00000000000000000000000000008001"||r.presentation_contract_revision!=="00000000000000000000000000001001"||r.resource_contract_revision!=="00000000000000000000000000006001")fail("contract pins");
const d=r.durable_authority;
if(d.family_identity!=="example.test/go-uab-11/state/application-v1"||d.state_owner!=="example.test/go-uab-11/state"||d.port_identity!=="example.test/go-uab-11/persistence/durable-port-v1"||d.port_owner!=="example.test/go-uab-11/persistence"||d.codec_identity!=="seme.durable-state.canonical.v1"||d.current_version!==2||d.maximum_payload_bytes!==65536||d.maximum_key_bytes!==256)fail("durable profile");
if(!hex32(d.current_type)||!hex32(d.key_type)||!hex32(d.domain_error_type)||JSON.stringify(d.versions.map(x=>x.number))!=="[1,2]"||d.versions.some(x=>!hex32(x.type)||!hex32(x.validator))||d.migrations.length!==1||d.migrations[0].from!==1||d.migrations[0].to!==2||!hex32(d.migrations[0].function))fail("durable types");
if(d.load.sequence!==0||d.load.identity!=="seme.storage.load.v1"||d.load.capability!=="seme.storage.load.v1.capability"||d.compare_exchange.sequence!==1||d.compare_exchange.identity!=="seme.storage.compare_exchange.v1"||d.compare_exchange.capability!=="seme.storage.compare_exchange.v1.capability")fail("durable operations");
const p=r.host_boundary_profile_probe;
if(p.family!==d.family_identity||p.load_identity!==d.load.identity||p.compare_exchange_identity!==d.compare_exchange.identity||p.load_sequence!==0||p.compare_exchange_sequence!==1||p.committed!==true)fail("host probe");
const aliases=Object.fromEntries(r.normalized_alias_authority.map(x=>[x.owner+"/"+x.stable_name,x]));
if(r.normalized_alias_authority.length!==2||r.source_bound_alias_authority.length!==2||!aliases["example.test/go-uab-11/application/Outcome"]||!aliases["example.test/go-uab-11/persistence/Plan"])fail("aliases");
for(const a of Object.values(aliases))if(!hex32(a.target_type)||![0,1].includes(a.visibility)||!Array.isArray(a.referenced_imports)||a.referenced_imports.length!==1)fail("alias shape");
const resources=Object.fromEntries(r.resources.map(x=>[x.identity,x]));
if(r.resources.length!==2||resources["example.test/go-uab-11/resource.notice.v1"]?.sha256!=="dde7c6f27c3a259a48b5c9e6b886a7f3c1c75f87c73699536eccb7149d1e7d48"||resources["example.test/go-uab-11/resource.notice.v1"]?.size!==26||resources["example.test/go-uab-11/resource.marker.v1"]?.sha256!=="d57b9b19e900f27112610f80812bf55d383d69ffd0e5796b7e1c84164ef15f9d"||resources["example.test/go-uab-11/resource.marker.v1"]?.size!==7)fail("resources");
if(!/^[0-9a-f]{64}$/.test(r.project_content_revision)||!/^[0-9a-f]{64}$/.test(r.durable_content_revision)||!/^[0-9a-f]{64}$/.test(r.source_bound_presentation_revision))fail("revisions");
NODE
reject_report() { name=$1 bundle=$2; if report "$bundle" > "$work/$name.out" 2> "$work/$name.err"; then echo "UPB-07 accepted $name" >&2; exit 1; fi; test ! -s "$work/$name.out"; }
for artifact in durable-state-v1.seme source-presentation-v1.seme project-v10.seme; do
  name=$(printf '%s' "$artifact" | tr '.-' '__')
  cp -R "$work/bundle-a" "$work/tamper-$name"; printf x >> "$work/tamper-$name/$artifact"; reject_report "tamper-$name" "$work/tamper-$name"
done
cp -R "$work/bundle-a" "$work/extra"; printf x > "$work/extra/extra"; reject_report extra "$work/extra"
cp -R "$work/bundle-a" "$work/missing"; rm "$work/missing/source-presentation-v1.seme"; reject_report missing "$work/missing"
ln -s "$work/bundle-a" "$work/bundle-link"; reject_report symlink "$work/bundle-link"
cp -R "$work/source" "$work/source-comment"
sed '1i\
// source-bound presentation revision adversary
' "$work/source-comment/persistence/planner.go" > "$work/planner-comment.go"
mv "$work/planner-comment.go" "$work/source-comment/persistence/planner.go"
build "$work/source-comment" 1 "$work/comment-bundle"
cp -R "$work/bundle-a" "$work/mixed"
cp "$work/comment-bundle/source-presentation-v1.seme" "$work/mixed/source-presentation-v1.seme"
mixed_digest=$(sha256sum "$work/mixed/source-presentation-v1.seme" | awk '{print $1}')
awk -v digest="$mixed_digest" 'NR==1 {print; next} $1=="source-presentation-v1.seme" {$2=digest} {print $1" "$2}' "$work/mixed/COMPLETE.sha256" > "$work/mixed-complete"
mv "$work/mixed-complete" "$work/mixed/COMPLETE.sha256"
reject_report mixed-presentation "$work/mixed"
rg -q 'project_v10|presentation' "$work/mixed-presentation.err"

# Class 7: projection preserves opaque bytes, remains native Go, and reaches an exact relift fixed point.
project "$work/source" "$work/bundle-a" "$work/projected-a"
grep -F 'type Plan = model.Result[Decision, int64]' "$work/projected-a/persistence/seme_projected.go" >/dev/null
for file in go.mod go.sum resources.json durable-selection.json resources/notice.txt resources/marker.bin; do cmp "$work/source/$file" "$work/projected-a/$file"; done
(cd "$work/projected-a" && GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./...)
build "$work/projected-a" 1 "$work/relift-a"
report "$work/relift-a" > "$work/relift-a-report.json"
node - "$work/report-a.json" "$work/relift-a-report.json" <<'NODE'
const fs=require("fs"), a=JSON.parse(fs.readFileSync(process.argv[2])), b=JSON.parse(fs.readFileSync(process.argv[3]));
const normalized=x=>({project_contract_revision:x.project_contract_revision,durable_contract_revision:x.durable_contract_revision,presentation_contract_revision:x.presentation_contract_revision,resource_contract_revision:x.resource_contract_revision,durable_authority:x.durable_authority,resources:x.resources,normalized_alias_authority:x.normalized_alias_authority,host_boundary_profile_probe:x.host_boundary_profile_probe});
const na=normalized(a), nb=normalized(b);
for(const key of Object.keys(na))if(JSON.stringify(na[key])!==JSON.stringify(nb[key]))throw Error(`normalized authority changed across projection: ${key}\noriginal=${JSON.stringify(na[key])}\nrelift=${JSON.stringify(nb[key])}`);
if(a.durable_content_revision===b.durable_content_revision)throw Error("source-bound durable revision did not reflect projected inventory");
if(a.source_bound_presentation_revision===b.source_bound_presentation_revision)throw Error("source-bound presentation revision did not reflect projected locations");
if(JSON.stringify(a.source_bound_alias_authority)===JSON.stringify(b.source_bound_alias_authority))throw Error("source-bound alias locations did not change");
NODE
cmp "$work/bundle-a/construction-v36.g1" "$work/relift-a/construction-v36.g1"
cmp "$work/bundle-a/execution-v36.seme" "$work/relift-a/execution-v36.seme"
project "$work/projected-a" "$work/relift-a" "$work/projected-b"
diff -ru "$work/projected-a" "$work/projected-b"
build "$work/projected-b" 1 "$work/relift-b"
diff -ru "$work/relift-a" "$work/relift-b"
report "$work/relift-b" > "$work/relift-b-report.json"
cmp "$work/relift-a-report.json" "$work/relift-b-report.json"
if project "$work/source" "$work/bundle-a" "$work/projected-a" > "$work/existing.out" 2> "$work/existing.err"; then echo 'UPB-07 overwrote projected destination' >&2; exit 1; fi
test ! -s "$work/existing.out"

echo 'Go UPB-07 seven-class authority gate passes'
