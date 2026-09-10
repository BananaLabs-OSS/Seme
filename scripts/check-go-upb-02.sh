#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb02.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
fixture="$repo/fixtures/go-upb02-package-v1"; module=example.test/go-project-build-v1; root_package="$module/application"
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}; pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001

(cd "$fixture" && GOTOOLCHAIN=local go test -count=1 -buildvcs=false ./...)
(cd "$repo/reference/go" &&
  go test -count=1 -buildvcs=false ./goprovider ./gopackageadapter ./packagedetail ./packagedetailinstance ./projectgraphinstance ./resolutionfidelity ./goprojectpipeline ./projectv3report &&
  go build -buildvcs=false -o "$work/build" ./cmd/go-upb02-build &&
  go build -buildvcs=false -o "$work/project" ./cmd/go-upb02-project &&
  go build -buildvcs=false -o "$work/report" ./cmd/project-v3-report &&
  go build -buildvcs=false -o "$work/observe" ./cmd/canonical-observe &&
  go build -buildvcs=false -o "$work/codec" ./cmd/pure-application-codec &&
  GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -buildvcs=false -o "$work/canonical-vm.wasm" ./cmd/canonical-wasm-cell)

build() {
  source=$1 revision=$2 destination=$3
  "$work/build" -project "$source" -module "$module" -package "$root_package" -entry Apply -revision "$revision" -out "$destination" \
    -execution-g1 "$repo/modules/execution/v35/module.g1" -execution-contract "$repo/modules/execution/v35/module.seme" \
    -package-v1 "$repo/modules/package/v1/module.seme" -package-v2 "$repo/modules/package/v2/module.seme" \
    -project-v1 "$repo/modules/project/v1/module.seme" -project-v2 "$repo/modules/project/v2/module.seme" -project-v3 "$repo/modules/project/v3/module.seme" \
    -k0 "$repo/bootstrap/seme-k0-linux-amd64" -g1-compiler "$repo/compiler/g1-compiler.k0"
}
report() {
  bundle=$1 output=$2
  "$work/report" --execution "$repo/modules/execution/v35/module.seme" --package-v1 "$repo/modules/package/v1/module.seme" --package-v2 "$repo/modules/package/v2/module.seme" \
    --project-v2-contract "$repo/modules/project/v2/module.seme" --project-v3-contract "$repo/modules/project/v3/module.seme" \
    --project "$bundle/project-v1.seme" --inventory "$bundle/inventory-v2.seme" --package-graph "$bundle/package-v2.seme" --composed "$bundle/project-v3.seme" > "$output"
}

# E1/E2/E3: authenticated resolution fidelity, exact package graph, and source-located ownership.
build "$fixture" 1 "$work/bundle-a"
build "$fixture" 99 "$work/bundle-b"
cmp "$work/bundle-a/project-v3.seme" "$work/bundle-b/project-v3.seme"
report "$work/bundle-a" "$work/report-a.json"
report "$work/bundle-a" "$work/report-b.json"
cmp "$work/report-a.json" "$work/report-b.json"
grep -q '"root":"example.test/go-project-build-v1/application"' "$work/report-a.json"
grep -q '"Visibility":"public"' "$work/report-a.json"
grep -q '"Visibility":"package"' "$work/report-a.json"
grep -q '"Name":"normalize"' "$work/report-a.json"
grep -q '"Alias":"model"' "$work/report-a.json"
grep -q '"Requested":"example.test/go-project-build-v1/model"' "$work/report-a.json"
grep -q '"Resolved":"example.test/go-project-build-v1/model"' "$work/report-a.json"
grep -q '"Class":"tracked"' "$work/report-a.json"
grep -q '"Preservation":"semantic-projection"' "$work/report-a.json"
test "$(grep -o '"Identity":"example.test/go-project-build-v1/' "$work/report-a.json" | wc -l)" -eq 3

# E4: native and canonical behavior agree with the checked request corpus.
"$work/observe" "$work/bundle-a/project-v1.seme" < "$fixture/vectors/requests.jsonl" > "$work/canonical.jsonl"
diff -u "$fixture/vectors/expected.jsonl" "$work/canonical.jsonl"

# E5: standalone Wasm and pinned Pulp realize the same canonical Program.
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

# E6: projection preserves the semantic Project exactly. Because Project v3
# intentionally commits source paths/digests, its honest byte-identity claim is
# that repeated lifts of the same projected source ignore client revision.
"$work/project" -root "$fixture" -construction "$work/bundle-a/construction.g1" -package-v2 "$work/bundle-a/package-v2.seme" -module "$module" -to "$work/projected"
(cd "$work/projected" && go test -count=1 -buildvcs=false ./...)
build "$work/projected" 7 "$work/bundle-relift-a"
build "$work/projected" 88 "$work/bundle-relift-b"
cmp "$work/bundle-a/project-v1.seme" "$work/bundle-relift-a/project-v1.seme"
cmp "$work/bundle-relift-a/project-v3.seme" "$work/bundle-relift-b/project-v3.seme"

# E7: missing imports, cycles, private access, tamper, and existing outputs fail closed.
reject_build() { source=$1 destination=$2; if build "$source" 3 "$destination" >"$destination.out" 2>"$destination.err"; then echo "UPB-02 accepted invalid project: $source" >&2; exit 1; fi; test ! -e "$destination"; }
cp -R "$fixture" "$work/missing"; sed 's|go-project-build-v1/policy|go-project-build-v1/missing|' "$work/missing/application/application.go" > "$work/missing/application/x"; mv "$work/missing/application/x" "$work/missing/application/application.go"; reject_build "$work/missing" "$work/reject-missing"
cp -R "$fixture" "$work/cycle"; sed '2i import _ "example.test/go-project-build-v1/application"' "$work/cycle/model/value.go" > "$work/cycle/model/x"; mv "$work/cycle/model/x" "$work/cycle/model/value.go"; reject_build "$work/cycle" "$work/reject-cycle"
cp -R "$fixture" "$work/private"; sed 's/model\.Normalize/model.normalize/' "$work/private/policy/policy.go" > "$work/private/policy/x"; mv "$work/private/policy/x" "$work/private/policy/policy.go"; reject_build "$work/private" "$work/reject-private"
cp "$work/bundle-a/project-v3.seme" "$work/tampered.seme"; printf x >> "$work/tampered.seme"
if "$work/report" --execution "$repo/modules/execution/v35/module.seme" --package-v1 "$repo/modules/package/v1/module.seme" --package-v2 "$repo/modules/package/v2/module.seme" --project-v2-contract "$repo/modules/project/v2/module.seme" --project-v3-contract "$repo/modules/project/v3/module.seme" --project "$work/bundle-a/project-v1.seme" --inventory "$work/bundle-a/inventory-v2.seme" --package-graph "$work/bundle-a/package-v2.seme" --composed "$work/tampered.seme" > "$work/tampered.out" 2> "$work/tampered.err"; then echo 'UPB-02 report accepted tamper' >&2; exit 1; fi
test ! -s "$work/tampered.out"
if build "$fixture" 4 "$work/bundle-a" > "$work/existing.out" 2> "$work/existing.err"; then echo 'UPB-02 overwrote output' >&2; exit 1; fi
test -f "$work/bundle-a/COMPLETE.sha256"

echo 'Go UPB-02: seven independent evidence classes pass'
