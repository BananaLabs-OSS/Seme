#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v38.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/cache"; export GOCACHE
(cd "$repo/reference/go" && go test -count=1 -buildvcs=false ./executionmodule)
(cd "$repo/reference/go" && go run -buildvcs=false ./cmd/execution-module-v38) > "$work/module.g1"
cmp "$repo/modules/execution/v38/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v38" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v38/module.seme" "$work/module.seme"
sed 's/en 0000000000000000000000000000a06c /en 0000000000000000000000000000a06b /' "$work/module.g1" > "$work/duplicate.g1"
if "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/duplicate.g1" "$work/bad.seme" >/dev/null 2>&1; then echo 'duplicate Evaluate declaration accepted' >&2; exit 1; fi
version=2
while [ "$version" -le 37 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done
echo 'Core Execution v38: Evaluate is additive and reproducible'
