#!/bin/sh
# Cross-artifact and boundary adversaries for the shared source-free authority.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd); prior=${1:?prior build export required}; result=${2:?result build export required}
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-upb12-adversaries.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM; GOCACHE="$work/go-cache"; export GOCACHE
reject(){ name=$1; root=$2; if "$repo/scripts/build-upb12-authority.sh" "$root" wasm32-pulp-upb12-host-v1 upb12 "$work/reject-$name" > "$work/$name.out" 2> "$work/$name.err"; then echo "UPB12 accepted $name adversary" >&2; exit 1; fi; test ! -e "$work/reject-$name"; test ! -s "$work/$name.out"; test -s "$work/$name.err"; }
copy(){ cp -R "$prior" "$work/$1"; }

copy dependency; printf x >> "$work/dependency/result-base/dependency-v1.seme"; reject dependency "$work/dependency"
copy resource; blob=$(find "$work/resource/result-base/blobs" -type f | sort | sed -n '1p'); printf x >> "$blob"; reject resource "$work/resource"
copy capability; node - "$work/capability/inputs/controlled-effects-selection.json" <<'NODE'
const fs=require("fs"),p=process.argv[2],x=JSON.parse(fs.readFileSync(p));x.effect.capability="ambient.network";fs.writeFileSync(p,JSON.stringify(x));
NODE
reject capability "$work/capability"
copy boundary; node - "$work/boundary/inputs/controlled-effects-selection.json" <<'NODE'
const fs=require("fs"),p=process.argv[2],x=JSON.parse(fs.readFileSync(p));x.bounds.maximum_effects=257;fs.writeFileSync(p,JSON.stringify(x));
NODE
reject boundary "$work/boundary"
copy graph; head -c 31 "$work/graph/result-base/construction-v36.g1" > "$work/graph/result-base/construction-v36.g1.next"; mv "$work/graph/result-base/construction-v36.g1.next" "$work/graph/result-base/construction-v36.g1"; node "$repo/reference/js/upb12-rewrite-bundle-manifest.mjs" "$work/graph/result-base"; reject graph "$work/graph"
copy revision; cp "$result/result-base/construction-v36.g1" "$work/revision/result-base/construction-v36.g1"; node "$repo/reference/js/upb12-rewrite-bundle-manifest.mjs" "$work/revision/result-base"; reject revision "$work/revision"
copy mixed; cp "$result/result-base/package-v4.seme" "$work/mixed/result-base/package-v4.seme"; node "$repo/reference/js/upb12-rewrite-bundle-manifest.mjs" "$work/mixed/result-base"; reject mixed-bundle "$work/mixed"
copy symlink; unlink "$work/symlink/result-base/dependency-v1.seme"; ln -s "$prior/result-base/dependency-v1.seme" "$work/symlink/result-base/dependency-v1.seme"; reject symlink "$work/symlink"
if "$repo/scripts/build-upb12-authority.sh" "$prior" wasm32-pulp-upb12-host-v1 upb12 "$work/existing" >/dev/null; then :; else exit 1; fi
if "$repo/scripts/build-upb12-authority.sh" "$prior" wasm32-pulp-upb12-host-v1 upb12 "$work/existing" > "$work/collision.out" 2> "$work/collision.err"; then echo 'UPB12 overwrote an authority destination' >&2; exit 1; fi
test ! -s "$work/collision.out"; test -s "$work/collision.err"
printf 'UPB12 adversaries: dependency, resource, revision, capability, bounds, graph, mixed bundle, symlink, and destination collision reject atomically\n'
