#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v101.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
if [ -z "${GOCACHE:-}" ]; then
  GOCACHE="$work/cache"
  export GOCACHE
fi
(cd "$repo/reference/go" && go test -count=1 -buildvcs=false ./executionmodule ./goprovider ./goprojector ./canonicaleval)
(cd "$repo/reference/go" && go run -buildvcs=false ./cmd/execution-module-v101) > "$work/module.g1"
cmp "$repo/modules/execution/v101/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v101" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v101/module.seme" "$work/module.seme"
(cd "$repo/reference/go" && go run -buildvcs=false ./cmd/go-session-proof \
  --module "$repo/modules/execution/v101/module.g1" \
  --project "$repo/fixtures/go-execution-v101" \
  --package example.test/go-execution-v101 --entry Wait \
  --out "$work/program.g1")
cmp "$repo/fixtures/go-execution-v101/program.g1" "$work/program.g1"
echo 'Core Execution v101: native channel selection is reproducible'
