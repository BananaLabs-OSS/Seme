#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v106.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
if [ -z "${GOCACHE:-}" ]; then
  GOCACHE="$work/cache"
  export GOCACHE
fi
(cd "$repo/reference/go" && go test -count=1 -buildvcs=false ./executionmodule ./goprovider ./goprojector ./canonicaleval)
(cd "$repo/reference/go" && go run -buildvcs=false ./cmd/execution-module-v106) > "$work/module.g1"
cmp "$repo/modules/execution/v106/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v106" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v106/module.seme" "$work/module.seme"
(cd "$repo/reference/go" && go run -buildvcs=false ./cmd/go-session-proof \
  --module "$repo/modules/execution/v106/module.g1" \
  --project "$repo/fixtures/go-execution-v106" \
  --package example.test/go-execution-v106 --entry Run \
  --out "$work/program.g1")
cmp "$repo/fixtures/go-execution-v106/program.g1" "$work/program.g1"
echo 'Core Execution v106: variadic native Go method calls are reproducible'
