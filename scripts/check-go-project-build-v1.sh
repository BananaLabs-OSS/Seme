#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-project-build-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; export GOCACHE

fixture="$repo/fixtures/go-project-build-v1"
(cd "$fixture" && go test -count=1 -buildvcs=false ./...)
(cd "$repo/reference/go" &&
  go test -count=1 -buildvcs=false ./projectemitter ./projectinstance ./packageinstance &&
  go build -buildvcs=false -o "$work/go-project-build" ./cmd/go-project-build &&
  go build -buildvcs=false -o "$work/project-snapshot-check" ./cmd/project-snapshot-check)

build() {
  source=$1
  revision=$2
  output=$3
  "$work/go-project-build" \
    --project "$source" \
    --module example.test/go-project-build-v1 \
    --package example.test/go-project-build-v1/application \
    --entry Apply \
    --revision "$revision" \
    --out "$output" \
    --execution-g1 "$repo/modules/execution/v35/module.g1" \
    --execution-contract "$repo/modules/execution/v35/module.seme" \
    --package-contract "$repo/modules/package/v1/module.seme" \
    --project-contract "$repo/modules/project/v1/module.seme" \
    --k0 "$repo/bootstrap/seme-k0-linux-amd64" \
    --g1-compiler "$repo/compiler/g1-compiler.k0" \
    --kernel-validator "$repo/compiler/kernel-wire-validator.k0"
  # The builder has already run the frozen compiler and Kernel validator over
  # the Session graph. The assembled project intentionally contains external
  # contract imports, so its authority is the typed Project/Package validators.
  "$work/project-snapshot-check" "$output"
}

build "$fixture" 1 "$work/revision-1.seme"
build "$fixture" 99 "$work/revision-99.seme"
cmp "$work/revision-1.seme" "$work/revision-99.seme"

# File discovery order, file names, and ordinary comments are presentation
# details. Reorder them without changing the typed Go program.
cp -R "$fixture" "$work/presentation"
mv "$work/presentation/model/value.go" "$work/presentation/model/z_value.go"
sed '/^\/\/ Normalize is/d' "$work/presentation/model/z_value.go" > "$work/presentation/model/z_value.tmp"
mv "$work/presentation/model/z_value.tmp" "$work/presentation/model/z_value.go"
sed '1a\
// Comments may move without changing the package semantics.\
// This line intentionally changes source presentation only.' \
  "$work/presentation/model/z_value.go" > "$work/presentation/model/a_value.go"
rm "$work/presentation/model/z_value.go"
build "$work/presentation" 7 "$work/presentation.seme"
cmp "$work/revision-1.seme" "$work/presentation.seme"

# A dependency-leaf behavior change must propagate through Package and Project
# revisions into different canonical artifact bytes.
cp -R "$fixture" "$work/semantic"
sed 's/return value + 1/return value + 2/' "$work/semantic/model/value.go" > "$work/semantic/model/value.tmp"
mv "$work/semantic/model/value.tmp" "$work/semantic/model/value.go"
build "$work/semantic" 2 "$work/semantic.seme"
if cmp -s "$work/revision-1.seme" "$work/semantic.seme"; then
  echo 'Go project build ignored a semantic dependency edit' >&2
  exit 1
fi

echo 'Go project build v1: native three-package closure, frozen compilation, validated project, revision/comment invariance, and semantic dependency invalidation pass'
