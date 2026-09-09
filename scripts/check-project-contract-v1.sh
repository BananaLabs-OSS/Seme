#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-project-contract.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; export GOCACHE

(cd "$repo/reference/go" && go test -count=1 -buildvcs=false ./projectmodule && go run -buildvcs=false ./cmd/project-module -out "$work/module.g1")
cmp "$repo/modules/project/v1/module.g1" "$work/module.g1"
(cd "$repo/modules/project/v1" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/module.g1" "$work/module.seme"
cmp "$repo/modules/project/v1/module.seme" "$work/module.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/modules/foundation/v1/validator.k0" "$work/module.seme"

# A module identity may repeat across revisions in one module family, but never
# across different semantic module families. Entity identities likewise may be
# revised within a family and may not be owned by two different families.
find "$repo/modules" -mindepth 3 -maxdepth 3 -name module.g1 -print | sort > "$work/modules.list"
audit_identities() { awk '
FNR==1 { split(FILENAME,p,"/"); family=p[length(p)-2] }
$1=="mo" { module=$2; if (moduleFamily[module]!="" && moduleFamily[module]!=family) { print "duplicate module identity " module > "/dev/stderr"; exit 1 } moduleFamily[module]=family }
$1=="rv" { revision=$2; if (revisionFamily[revision]!="" && revisionFamily[revision]!=family) { print "duplicate revision identity " revision > "/dev/stderr"; exit 1 } revisionFamily[revision]=family }
$1=="en" { id=$2; if (entityFamily[id]!="" && entityFamily[id]!=family) { print "duplicate entity identity " id > "/dev/stderr"; exit 1 } entityFamily[id]=family }
' "$@"; }
audit_identities $(cat "$work/modules.list")
mkdir -p "$work/project/v1"
sed 's/0000000000000000000000000000e000/0000000000000000000000000000c000/g' "$work/module.g1" > "$work/project/v1/module.g1"
if audit_identities $(cat "$work/modules.list") "$work/project/v1/module.g1" 2>/dev/null; then
  echo 'identity audit accepted prior c000 collision' >&2
  exit 1
fi

echo 'Project Contract v1: exact imports, fresh identities, deterministic generation, canonical validation, and repository collision audit passed'
