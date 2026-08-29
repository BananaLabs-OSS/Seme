#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-modules.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

k0="$repo/bootstrap/seme-k0-linux-amd64"
g1="$repo/compiler/g1-compiler.k0"
kernel="$repo/compiler/kernel-wire-validator-located.k0"
patch="$repo/modules/patch/v1"
foundation="$repo/modules/foundation/v1"

(
    cd "$foundation"
    sha256sum -c module.g1.sha256
    sha256sum -c module.seme.sha256
)

(cd "$repo/reference/go" && go run ./cmd/foundation-module) > "$work/foundation-v1.g1"
cmp "$foundation/module.g1" "$work/foundation-v1.g1"
"$k0" "$g1" "$work/foundation-v1.g1" "$work/foundation-v1.seme"
cmp "$foundation/module.seme" "$work/foundation-v1.seme"
"$k0" "$kernel" "$work/foundation-v1.seme" "$work/foundation-v1.validated.seme"
cmp "$work/foundation-v1.seme" "$work/foundation-v1.validated.seme"

(
    cd "$patch"
    sha256sum -c module.g1.sha256
    sha256sum -c module.seme.sha256
)

"$k0" "$g1" "$patch/module.g1" "$work/patch-v1.seme"
cmp "$patch/module.seme" "$work/patch-v1.seme"
"$k0" "$kernel" "$work/patch-v1.seme" "$work/patch-v1.validated.seme"
cmp "$work/patch-v1.seme" "$work/patch-v1.validated.seme"

(cd "$repo/reference/go" && go test ./...)

echo "Semantic modules: Foundation v1 and Patch v1 canonical graphs and reference conformance passed"
