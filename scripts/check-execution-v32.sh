#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-execution-v32.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

(cd "$repo/reference/go" && go test -buildvcs=false ./executionmodule && go run -buildvcs=false ./cmd/execution-module-v32 > "$work/module.g1")
cmp "$repo/modules/execution/v32/module.g1" "$work/module.g1"
(cd "$repo/modules/execution/v32" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/execution/v32/module.seme" "$work/module.seme"

sed 's/en 000000000000000000000000000a0651 /en 000000000000000000000000000a0650 /' "$work/module.g1" > "$work/duplicate-field.g1"
if "$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/duplicate-field.g1" "$work/duplicate-field.seme" >/dev/null 2>&1; then
  echo "duplicate v32 field identity accepted" >&2
  exit 1
fi

version=2
while [ "$version" -le 31 ]; do
  (cd "$repo/reference/go" && go run -buildvcs=false "./cmd/execution-module-v$version") > "$work/v$version.g1"
  cmp "$repo/modules/execution/v$version/module.g1" "$work/v$version.g1"
  version=$((version + 1))
done

echo "Core Execution v32: total variant matches and exact opaque bytes are additive and reproducible"
