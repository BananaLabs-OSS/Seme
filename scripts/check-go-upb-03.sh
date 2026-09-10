#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb03.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
fixture="$repo/fixtures/go-upb03-dependency-v1"
proxy="$repo/fixtures/go-upb03-offline-proxy"
module=example.test/go-upb03-dependency-v1
root_package="$module/application"
dependency=example.test/seme/checksum
version=v1.2.3
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}; pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001

# E1: the checked vectors execute natively using only the pinned offline proxy.
(cd "$fixture" && GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./...)

(cd "$repo/reference/go" &&
  go test -count=1 -buildvcs=false ./godependencyresolver ./goofflineclosure ./dependencyresolution ./dependencyemitter ./dependencyinstance ./dependencyreport ./goprojectdependency ./goupb03pipeline ./projectdependencyinstance ./projectv4emitter ./projectv4report &&
  go build -buildvcs=false -o "$work/build" ./cmd/go-upb03-build &&
  go build -buildvcs=false -o "$work/dependency-report" ./cmd/dependency-report &&
  go build -buildvcs=false -o "$work/project-report" ./cmd/project-v4-report &&
  go build -buildvcs=false -o "$work/project" ./cmd/go-upb02-project &&
  go build -buildvcs=false -o "$work/observe" ./cmd/canonical-observe &&
  go build -buildvcs=false -o "$work/codec" ./cmd/pure-application-codec &&
  GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -buildvcs=false -o "$work/canonical-vm.wasm" ./cmd/canonical-wasm-cell)

build() {
  source=$1 proxy_root=$2 revision=$3 destination=$4
  GOCACHE="$destination.go-cache" "$work/build" -project "$source" -proxy "$proxy_root" -module "$module" -package "$root_package" -entry Apply -revision "$revision" -out "$destination" \
    -dependency "$dependency" -version "$version" -local-from "$root_package" -local-to "$module/model" \
    -execution-g1 "$repo/modules/execution/v35/module.g1" -execution-contract "$repo/modules/execution/v35/module.seme" \
    -package-v1 "$repo/modules/package/v1/module.seme" -package-v2 "$repo/modules/package/v2/module.seme" \
    -project-v1 "$repo/modules/project/v1/module.seme" -project-v2 "$repo/modules/project/v2/module.seme" -project-v3 "$repo/modules/project/v3/module.seme" -project-v4 "$repo/modules/project/v4/module.seme" \
    -dependency-v1 "$repo/modules/dependency/v1/module.seme" -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0"
}
dependency_report() { "$work/dependency-report" --contract "$repo/modules/dependency/v1/module.seme" --instance "$1/dependency-v1.seme" > "$2"; }
project_report() {
  b=$1 out=$2
  "$work/project-report" --execution "$repo/modules/execution/v35/module.seme" --package-v1 "$repo/modules/package/v1/module.seme" --package-v2 "$repo/modules/package/v2/module.seme" --dependency-contract "$repo/modules/dependency/v1/module.seme" \
    --project-v2-contract "$repo/modules/project/v2/module.seme" --project-v3-contract "$repo/modules/project/v3/module.seme" --project-v4-contract "$repo/modules/project/v4/module.seme" \
    --project "$b/project-v1.seme" --inventory "$b/inventory-v2.seme" --package-graph "$b/package-v2.seme" --project-v3 "$b/project-v3.seme" --dependency "$b/dependency-v1.seme" --composed "$b/project-v4.seme" > "$out"
}

# E2/E3/E4: exact offline resolution is repeatable, authenticated into Project
# v4, and independently inspectable without source bytes.
build "$fixture" "$proxy" 1 "$work/bundle-a"
build "$fixture" "$proxy" 99 "$work/bundle-b"
cmp "$work/bundle-a/dependency-v1.seme" "$work/bundle-b/dependency-v1.seme"
cmp "$work/bundle-a/project-v4.seme" "$work/bundle-b/project-v4.seme"
dependency_report "$work/bundle-a" "$work/dependency-a.json"
dependency_report "$work/bundle-a" "$work/dependency-b.json"
cmp "$work/dependency-a.json" "$work/dependency-b.json"
grep -q '"contract_revision":"0000000000000000000000000000f001"' "$work/dependency-a.json"
grep -q '"identity":"example.test/seme/checksum"' "$work/dependency-a.json"
grep -q '"version":"v1.2.3"' "$work/dependency-a.json"
grep -q '"integrity":"h1:iqNHCyOnAuyQDdyru3Tt6tsFgwUzMObOH5iQ9+5SMXA="' "$work/dependency-a.json"
project_report "$work/bundle-a" "$work/project-a.json"
project_report "$work/bundle-a" "$work/project-b.json"
cmp "$work/project-a.json" "$work/project-b.json"
grep -q '"contract_revision":"0000000000000000000000000000e004"' "$work/project-a.json"

# E5: canonical, standalone Wasm, and the pinned Pulp runtime agree.
"$work/observe" "$work/bundle-a/project-v1.seme" < "$fixture/vectors/requests.jsonl" > "$work/canonical.jsonl"
diff -u "$fixture/vectors/expected.jsonl" "$work/canonical.jsonl"
"$work/codec" encode "$work/bundle-a/project-v1.seme" < "$fixture/vectors/requests.jsonl" > "$work/requests.hex"
node "$repo/reference/js/canonical-wasm-cell-runner.mjs" "$work/canonical-vm.wasm" "$work/bundle-a/project-v1.seme" "$work/requests.hex" > "$work/wasm.hex"
"$work/codec" observe "$work/bundle-a/project-v1.seme" < "$work/wasm.hex" > "$work/wasm.jsonl"
diff -u "$fixture/vectors/expected.jsonl" "$work/wasm.jsonl"
test -f "$pulp_repo/go.mod"; git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pinned" "$work/pulp"; git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pinned"
mkdir -p "$work/pinned/cmd/pulp-seme-canonical-vm-proof"; cp "$repo/targets/wasm/pulp-canonical-vm-v1/runner.go" "$work/pinned/cmd/pulp-seme-canonical-vm-proof/main.go"
(cd "$work/pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-canonical-vm-proof)
cp "$repo/targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"; cp "$work/canonical-vm.wasm" "$work/pulp/canonical-vm.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -graph "$work/bundle-a/project-v1.seme" -requests "$work/requests.hex" > "$work/pulp.raw.jsonl"
node "$repo/reference/js/pulp-canonical-vm-output.mjs" "$work/pulp.raw.jsonl" > "$work/pulp.hex"
"$work/codec" observe "$work/bundle-a/project-v1.seme" < "$work/pulp.hex" > "$work/pulp.jsonl"
diff -u "$fixture/vectors/expected.jsonl" "$work/pulp.jsonl"

# E6: strict projection builds natively and re-lifts to identical semantic and
# dependency artifacts; client revisions do not enter canonical identity.
"$work/project" -root "$fixture" -construction "$work/bundle-a/construction.g1" -package-v2 "$work/bundle-a/package-v2.seme" -module "$module" -to "$work/projected"
cmp "$fixture/go.mod" "$work/projected/go.mod"
cmp "$fixture/go.sum" "$work/projected/go.sum"
(cd "$work/projected" && GOTOOLCHAIN=local GOPROXY="file://$proxy" GOSUMDB=off go test -count=1 -buildvcs=false ./...)
build "$work/projected" "$proxy" 7 "$work/relift-a"
build "$work/projected" "$proxy" 88 "$work/relift-b"
cmp "$work/bundle-a/project-v1.seme" "$work/relift-a/project-v1.seme"
cmp "$work/relift-a/dependency-v1.seme" "$work/relift-b/dependency-v1.seme"
cmp "$work/relift-a/project-v4.seme" "$work/relift-b/project-v4.seme"
dependency_report "$work/relift-a" "$work/dependency-relift.json"
grep -q '"identity":"example.test/seme/checksum"' "$work/dependency-relift.json"
grep -q '"version":"v1.2.3"' "$work/dependency-relift.json"
grep -q '"integrity":"h1:iqNHCyOnAuyQDdyru3Tt6tsFgwUzMObOH5iQ9+5SMXA="' "$work/dependency-relift.json"
if cmp -s "$work/bundle-a/dependency-v1.seme" "$work/relift-a/dependency-v1.seme"; then
  echo 'UPB-03 projection erased changed local source attestation' >&2; exit 1
fi

# E7: every unsupported or unauthenticated dependency state fails closed.
reject_build() { src=$1 px=$2 dest=$3; if build "$src" "$px" 3 "$dest" >"$dest.out" 2>"$dest.err";then echo "UPB-03 accepted invalid input: $src" >&2;exit 1;fi;test ! -e "$dest";test ! -s "$dest.out"; }
cp -R "$fixture" "$work/floating"; sed 's/v1\.2\.3/latest/' "$work/floating/go.mod" > "$work/floating/go.mod.x"; mv "$work/floating/go.mod.x" "$work/floating/go.mod"; reject_build "$work/floating" "$proxy" "$work/reject-floating"
cp -R "$fixture" "$work/replace"; printf '\nreplace example.test/seme/checksum => ../x\n' >> "$work/replace/go.mod"; reject_build "$work/replace" "$proxy" "$work/reject-replace"
cp -R "$fixture" "$work/undeclared"; sed '/^require /d' "$work/undeclared/go.mod" > "$work/undeclared/go.mod.x"; mv "$work/undeclared/go.mod.x" "$work/undeclared/go.mod"; reject_build "$work/undeclared" "$proxy" "$work/reject-undeclared"
cp -R "$fixture" "$work/badsum"; sed 's/h1:[A-Za-z0-9+\/=]*/h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA/' "$work/badsum/go.sum" > "$work/badsum/go.sum.x"; mv "$work/badsum/go.sum.x" "$work/badsum/go.sum"; reject_build "$work/badsum" "$proxy" "$work/reject-badsum"
cp -R "$proxy" "$work/badproxy"; truncate -s -1 "$work/badproxy/example.test/seme/checksum/@v/v1.2.3.zip"; reject_build "$fixture" "$work/badproxy" "$work/reject-zip"
cp -R "$fixture" "$work/symlink"; mv "$work/symlink/model/value.go" "$work/value.go"; ln -s "$work/value.go" "$work/symlink/model/value.go"; reject_build "$work/symlink" "$proxy" "$work/reject-symlink"
cp "$work/bundle-a/project-v4.seme" "$work/tampered.seme"; printf x >> "$work/tampered.seme"
# Directly substitute the tampered composed graph and require empty stdout.
if "$work/project-report" --execution "$repo/modules/execution/v35/module.seme" --package-v1 "$repo/modules/package/v1/module.seme" --package-v2 "$repo/modules/package/v2/module.seme" --dependency-contract "$repo/modules/dependency/v1/module.seme" --project-v2-contract "$repo/modules/project/v2/module.seme" --project-v3-contract "$repo/modules/project/v3/module.seme" --project-v4-contract "$repo/modules/project/v4/module.seme" --project "$work/bundle-a/project-v1.seme" --inventory "$work/bundle-a/inventory-v2.seme" --package-graph "$work/bundle-a/package-v2.seme" --project-v3 "$work/bundle-a/project-v3.seme" --dependency "$work/bundle-a/dependency-v1.seme" --composed "$work/tampered.seme" > "$work/graph-tamper.out" 2> "$work/graph-tamper.err";then echo 'UPB-03 accepted graph tamper' >&2;exit 1;fi
test ! -s "$work/graph-tamper.out"
if build "$fixture" "$proxy" 4 "$work/bundle-a" > "$work/existing.out" 2> "$work/existing.err";then echo 'UPB-03 overwrote output' >&2;exit 1;fi
test -f "$work/bundle-a/COMPLETE.sha256"

echo 'Go UPB-03: seven independent evidence classes pass'
