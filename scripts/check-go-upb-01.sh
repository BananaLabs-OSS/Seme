#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb01.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
fixture="$repo/fixtures/go-project-build-v1"
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001

(cd "$fixture" && go test -count=1 -buildvcs=false ./...)
(cd "$repo/reference/go" &&
  go test -count=1 -buildvcs=false ./projectsource ./projectbundle ./projectroundtrip ./sourceinventory &&
  go build -buildvcs=false -o "$work/project-build" ./cmd/go-project-build &&
  go build -buildvcs=false -o "$work/source-proof" ./cmd/go-upb01-source &&
  go build -buildvcs=false -o "$work/observe" ./cmd/canonical-observe &&
  go build -buildvcs=false -o "$work/codec" ./cmd/pure-application-codec &&
  GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -buildvcs=false -o "$work/canonical-vm.wasm" ./cmd/canonical-wasm-cell)

build_project() {
  source=$1; revision=$2; output=$3
  "$work/project-build" --project "$source" \
    --module example.test/go-project-build-v1 \
    --package example.test/go-project-build-v1/application --entry Apply \
    --revision "$revision" --out "$output" \
    --execution-g1 "$repo/modules/execution/v35/module.g1" \
    --execution-contract "$repo/modules/execution/v35/module.seme" \
    --package-contract "$repo/modules/package/v1/module.seme" \
    --project-contract "$repo/modules/project/v1/module.seme" \
    --k0 "$repo/bootstrap/seme-k0-linux-amd64" \
    --g1-compiler "$repo/compiler/g1-compiler.k0" \
    --kernel-validator "$repo/compiler/kernel-wire-validator.k0"
}
inventory() {
  source=$1; project=$2; output=$3; copy=$4
  "$work/source-proof" --root "$source" --project "$project" --out "$output" \
    --copy "$copy" --execution-contract "$repo/modules/execution/v35/module.seme" \
    --package-contract "$repo/modules/package/v1/module.seme" \
    --project-contract "$repo/modules/project/v2/module.seme"
}

build_project "$fixture" 1 "$work/project.seme"
"$work/observe" "$work/project.seme" < "$fixture/vectors/requests.jsonl" > "$work/observed.jsonl"
diff -u "$fixture/vectors/expected.jsonl" "$work/observed.jsonl"

inventory "$fixture" "$work/project.seme" "$work/inventory-a.seme" "$work/projected-a"
inventory "$fixture" "$work/project.seme" "$work/inventory-b.seme" "$work/projected-b"
cmp "$work/inventory-a.seme" "$work/inventory-b.seme"
diff -r "$fixture" "$work/projected-a"
diff -r "$work/projected-a" "$work/projected-b"

# The exact projected native tree remains independently buildable and re-lifts
# to precisely the same semantic Project artifact at another client revision.
(cd "$work/projected-a" && go test -count=1 -buildvcs=false ./...)
build_project "$work/projected-a" 99 "$work/relifted.seme"
cmp "$work/project.seme" "$work/relifted.seme"

"$work/codec" encode "$work/project.seme" < "$fixture/vectors/requests.jsonl" > "$work/requests.hex"
node "$repo/reference/js/canonical-wasm-cell-runner.mjs" "$work/canonical-vm.wasm" "$work/project.seme" "$work/requests.hex" > "$work/wasm.hex"
"$work/codec" observe "$work/project.seme" < "$work/wasm.hex" > "$work/wasm.jsonl"
diff -u "$fixture/vectors/expected.jsonl" "$work/wasm.jsonl"
test -f "$pulp_repo/go.mod"
git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}"
mkdir "$work/pinned" "$work/pulp"
git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pinned"
mkdir -p "$work/pinned/cmd/pulp-seme-canonical-vm-proof"
cp "$repo/targets/wasm/pulp-canonical-vm-v1/runner.go" "$work/pinned/cmd/pulp-seme-canonical-vm-proof/main.go"
(cd "$work/pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-canonical-vm-proof)
cp "$repo/targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"
cp "$work/canonical-vm.wasm" "$work/pulp/canonical-vm.wasm"
"$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -graph "$work/project.seme" -requests "$work/requests.hex" > "$work/pulp.raw.jsonl"
node "$repo/reference/js/pulp-canonical-vm-output.mjs" "$work/pulp.raw.jsonl" > "$work/pulp.hex"
"$work/codec" observe "$work/project.seme" < "$work/pulp.hex" > "$work/pulp.jsonl"
diff -u "$fixture/vectors/expected.jsonl" "$work/pulp.jsonl"

# An invalid filesystem state rejects before publishing either inventory or
# round-trip destination.
cp -R "$fixture" "$work/invalid"
ln -s NOTICE "$work/invalid/alias"
if inventory "$work/invalid" "$work/project.seme" "$work/rejected.seme" "$work/rejected-copy" >"$work/rejected.out" 2>"$work/rejected.err"; then
  echo 'Go UPB-01 accepted a symlink source unit' >&2; exit 1
fi
rg -q 'project_source.symlink' "$work/rejected.err"
test ! -e "$work/rejected.seme"
test ! -e "$work/rejected-copy"

# Reuse the authoritative semantic-build invariance proof.
"$repo/scripts/check-go-project-build-v1.sh"

echo 'Go UPB-01 foundation: native project, canonical inventory/lift, behavior target parity, preserved-region copy, and atomic symlink rejection pass'
echo 'Go UPB-01 score remains unclaimed: semantic source projection, independent fidelity evidence, and the complete atomic adversary set remain'
