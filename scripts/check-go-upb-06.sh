#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb06.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
proxy="$repo/fixtures/go-upb03-offline-proxy"
selection="$repo/fixtures/go-upb05-configuration-overlay/configuration-selection.json"

"$repo/scripts/check-go-upb-06-fixture.sh"
"$repo/scripts/check-go-upb-06-runtime.sh"
"$repo/scripts/materialize-go-upb06-fixture.sh" "$work/source"
(cd "$repo/reference/go" &&
  go test -count=1 ./goupb06pipeline ./goupb06bundle ./resourceinstance ./projectv9instance ./cmd/go-upb06-build ./cmd/go-upb06-project ./cmd/go-upb06-report &&
  go build -buildvcs=false -o "$work/build" ./cmd/go-upb06-build &&
  go build -buildvcs=false -o "$work/project" ./cmd/go-upb06-project &&
  go build -buildvcs=false -o "$work/report" ./cmd/go-upb06-report)

build() {
  source=$1 revision=$2 destination=$3 manifest=$4
  "$work/build" -project "$source" -proxy "$proxy" -module example.test/go-uab-11 -package example.test/go-uab-11/service -entry ApplyConfiguredResource -revision "$revision" -out "$destination" \
    -dependency example.test/seme/checksum -version v1.2.3 -local-from example.test/go-uab-11/application -local-to example.test/go-uab-11/model -selection "$selection" -resources "$manifest" -resource-owner example.test/go-uab-11/service \
    -execution-g1 "$repo/modules/execution/v36/module.g1" -execution-contract "$repo/modules/execution/v36/module.seme" -foundation-contract "$repo/modules/foundation/v1/module.seme" \
    -package-v4 "$repo/modules/package/v4/module.seme" -project-v8 "$repo/modules/project/v8/module.seme" -dependency-v1 "$repo/modules/dependency/v1/module.seme" -configuration-v3 "$repo/modules/configuration/v3/module.seme" \
    -resource-v1 "$repo/modules/resource/v1/module.seme" -project-v9 "$repo/modules/project/v9/module.seme" -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0"
}
authority_args() {
  printf '%s\n' -selection "$selection" -foundation "$repo/modules/foundation/v1/module.seme" -execution "$repo/modules/execution/v36/module.seme" -package "$repo/modules/package/v4/module.seme" -dependency "$repo/modules/dependency/v1/module.seme" -configuration "$repo/modules/configuration/v3/module.seme" -resource "$repo/modules/resource/v1/module.seme" -project-v8 "$repo/modules/project/v8/module.seme" -project-v9 "$repo/modules/project/v9/module.seme" -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0"
}
report() {
  bundle=$1
  set -- $(authority_args)
  "$work/report" -bundle "$bundle" "$@"
}
project() {
  root=$1 bundle=$2 destination=$3
  set -- $(authority_args)
  "$work/project" -root "$root" -to "$destination" -module example.test/go-uab-11 -bundle "$bundle" "$@"
}

# Identical offline inputs reproduce every canonical artifact, detached blob,
# completion manifest, and deterministic source-free authority report.
build "$work/source" 1 "$work/bundle-a" "$work/source/resources.json"
build "$work/source" 1 "$work/bundle-b" "$work/source/resources.json"
diff -ru "$work/bundle-a" "$work/bundle-b"
report "$work/bundle-a" > "$work/report-a.json"
report "$work/bundle-a" > "$work/report-b.json"
cmp "$work/report-a.json" "$work/report-b.json"
node -e '
const r=JSON.parse(require("fs").readFileSync(process.argv[1]));
if(r.project_contract_revision!=="0000000000000000000000000000e00b"||r.resource_contract_revision!=="00000000000000000000000000006001"||r.snapshot_revision.length!==64||r.resources.length!==2)throw Error("authority");
const by=Object.fromEntries(r.resources.map(x=>[x.identity,x]));
if(by["example.test/go-uab-11/resource.notice.v1"].sha256!=="dde7c6f27c3a259a48b5c9e6b886a7f3c1c75f87c73699536eccb7149d1e7d48"||by["example.test/go-uab-11/resource.notice.v1"].size!==26||by["example.test/go-uab-11/resource.notice.v1"].destination!=="resources/notice.txt")throw Error("notice");
if(by["example.test/go-uab-11/resource.marker.v1"].sha256!=="d57b9b19e900f27112610f80812bf55d383d69ffd0e5796b7e1c84164ef15f9d"||by["example.test/go-uab-11/resource.marker.v1"].size!==7||by["example.test/go-uab-11/resource.marker.v1"].destination!=="resources/marker.bin")throw Error("marker");
' "$work/report-a.json"

# Projection is authenticated and create-only. Opaque project data and both
# detached resource byte streams survive exactly; projected Go remains native.
project "$work/source" "$work/bundle-a" "$work/projected"
for file in go.mod go.sum resources.json resources/notice.txt resources/marker.bin application/application_test.go service/service_test.go service/native_corpus_test.go service/native_resource_corpus_test.go; do cmp "$work/source/$file" "$work/projected/$file"; done
(cd "$work/projected" && GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./...)
build "$work/projected" 1 "$work/relift" "$work/projected/resources.json"
cmp "$work/bundle-a/execution-v36.seme" "$work/relift/execution-v36.seme"
build "$work/projected" 1 "$work/relift-repeat" "$work/projected/resources.json"
diff -ru "$work/relift" "$work/relift-repeat"
report "$work/relift" > "$work/relift-report.json"
node -e '
const fs=require("fs"),a=JSON.parse(fs.readFileSync(process.argv[1])),b=JSON.parse(fs.readFileSync(process.argv[2]));
if(JSON.stringify(a.resources)!==JSON.stringify(b.resources)||a.project_contract_revision!==b.project_contract_revision||a.resource_contract_revision!==b.resource_contract_revision)throw Error("relift resource authority");
' "$work/report-a.json" "$work/relift-report.json"
if project "$work/source" "$work/bundle-a" "$work/projected" > "$work/existing.out" 2> "$work/existing.err"; then echo 'UPB-06 overwrote projected destination' >&2; exit 1; fi
test ! -s "$work/existing.out"

# Strict declaration and source acquisition adversaries reject atomically.
for adversary in traversal missing digest duplicate-destination over-limit unknown-field; do
  if build "$work/source" 2 "$work/reject-$adversary" "$repo/fixtures/go-upb06-resource-overlay/adversaries/$adversary.json" > "$work/$adversary.out" 2> "$work/$adversary.err"; then echo "UPB-06 accepted $adversary" >&2; exit 1; fi
  test ! -e "$work/reject-$adversary"; test ! -s "$work/$adversary.out"
done
cp -R "$work/source" "$work/symlink-source"
rm "$work/symlink-source/resources/notice.txt"
ln -s "$repo/fixtures/go-upb06-resource-overlay/adversaries/symlink-target.txt" "$work/symlink-source/resources/notice.txt"
if build "$work/symlink-source" 2 "$work/reject-symlink" "$work/symlink-source/resources.json" > "$work/symlink.out" 2> "$work/symlink.err"; then echo 'UPB-06 accepted resource symlink' >&2; exit 1; fi
test ! -e "$work/reject-symlink"; test ! -s "$work/symlink.out"

# Source-free loader rejects blob, manifest, artifact, mix-and-match, orphan,
# and directory-shape tampering without emitting an authority report.
reject_report() {
  name=$1 bundle=$2
  if report "$bundle" > "$work/$name.out" 2> "$work/$name.err"; then echo "UPB-06 accepted $name" >&2; exit 1; fi
  test ! -s "$work/$name.out"
}
cp -R "$work/bundle-a" "$work/tamper-blob"; printf x >> "$work/tamper-blob/blobs/d57b9b19e900f27112610f80812bf55d383d69ffd0e5796b7e1c84164ef15f9d"; reject_report tamper-blob "$work/tamper-blob"
cp -R "$work/bundle-a" "$work/tamper-resource"; printf x >> "$work/tamper-resource/resource-v1.seme"; reject_report tamper-resource "$work/tamper-resource"
cp -R "$work/bundle-a" "$work/tamper-project"; printf x >> "$work/tamper-project/project-v9.seme"; reject_report tamper-project "$work/tamper-project"
cp -R "$work/bundle-a" "$work/orphan"; printf x > "$work/orphan/blobs/2d711642b726b04401627ca9fbac32f5da7e5ee2ac9175e3cce89b73a6fbdc5a"; reject_report orphan "$work/orphan"
cp -R "$work/bundle-a" "$work/extra"; printf x > "$work/extra/extra"; reject_report extra "$work/extra"
ln -s "$work/bundle-a" "$work/bundle-link"; reject_report symlink-bundle "$work/bundle-link"
cp -R "$work/source" "$work/changed"; printf 'Seme resources — changed\n' > "$work/changed/resources/notice.txt"; build "$work/changed" 2 "$work/bundle-changed" "$work/changed/resources.json" > "$work/changed.out" 2> "$work/changed.err" && { echo 'UPB-06 accepted changed bytes under old digest' >&2; exit 1; }; test ! -e "$work/bundle-changed"

echo 'Go UPB-06 authority evidence passes over 2,066 cumulative resource cases'
