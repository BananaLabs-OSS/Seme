#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v89.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/cache"; export GOCACHE
(cd "$repo/reference/go" && go test -count=1 -buildvcs=false ./executionmodule ./goprovider ./goprojector ./canonicaleval)
(cd "$repo/reference/go" && go run -buildvcs=false ./cmd/execution-module-v89) > "$work/module.g1"
cmp "$repo/modules/execution/v89/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v89" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v89/module.seme" "$work/module.seme"
(cd "$repo/reference/go" && go run -buildvcs=false ./cmd/go-session-proof \
  --module "$repo/modules/execution/v89/module.g1" \
  --project "$repo/fixtures/go-execution-v89" \
  --package example.test/go-execution-v89 --entry Expand \
  --out "$work/program.g1")
cmp "$repo/fixtures/go-execution-v89/program.g1" "$work/program.g1"
version=2
while [ "$version" -le 70 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo 'Core Execution v89: typed native variadic expansion is reproducible'
