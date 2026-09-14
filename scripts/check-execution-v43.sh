#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v43.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/cache"; export GOCACHE
(cd "$repo/reference/go" && go test -count=1 -buildvcs=false ./executionmodule ./goprovider ./canonicaleval)
(cd "$repo/reference/go" && go run -buildvcs=false ./cmd/execution-module-v43) > "$work/module.g1"
cmp "$repo/modules/execution/v43/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v43" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v43/module.seme" "$work/module.seme"
version=2
while [ "$version" -le 42 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo 'Core Execution v43: typed native field observation is additive and reproducible'
