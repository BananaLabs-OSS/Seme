#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-language-service-v1.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

module="$repo/modules/language-service/v1"
k0="$repo/bootstrap/seme-k0-linux-amd64"
g1="$repo/compiler/g1-compiler.k0"
kernel="$repo/compiler/kernel-wire-validator.k0"
foundation="$repo/modules/foundation/v1/validator.k0"

(
    cd "$repo/reference/go"
    go test -buildvcs=false ./languageservice
    go run -buildvcs=false ./cmd/language-service-module > "$work/module.g1"
)
cmp "$module/module.g1" "$work/module.g1"
(
    cd "$module"
    sha256sum -c module.g1.sha256
    sha256sum -c module.seme.sha256
)
"$k0" "$g1" "$work/module.g1" "$work/module.seme"
cmp "$module/module.seme" "$work/module.seme"
"$k0" "$kernel" "$work/module.seme"
"$k0" "$foundation" "$work/module.seme"

echo "Live Language Service v1: deterministic schema and stale-update semantics passed"
