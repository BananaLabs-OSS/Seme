#!/bin/sh
set -eu

if [ "$#" -ne 2 ]; then
    echo "usage: apply-patch-v1.sh WORKSPACE OUTPUT" >&2
    exit 64
fi

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
workspace=$1
output=$2
output_dir=$(dirname -- "$output")
output_name=$(basename -- "$output")
stage=$(mktemp -d "$output_dir/.$output_name.patch-v1.XXXXXX")
trap 'rm -rf "$stage"' EXIT HUP INT TERM

k0="$repo/bootstrap/seme-k0-linux-amd64"
kernel="$repo/compiler/kernel-wire-validator.k0"
foundation="$repo/modules/foundation/v1/validator.k0"
patch="$repo/modules/patch/v1"
sha256="$repo/modules/digest/sha256/v1/digest.k0"

"$k0" "$kernel" "$workspace"
"$k0" "$foundation" "$workspace"
"$k0" "$patch/apply-candidate.k0" "$workspace" "$stage/candidate.seme"
"$k0" "$kernel" "$stage/candidate.seme"
"$k0" "$foundation" "$stage/candidate.seme"
"$k0" "$patch/revision-transcript.k0" "$workspace" \
    "$stage/candidate.seme" "$stage/transcript"
"$k0" "$sha256" "$stage/transcript" "$stage/digest"
"$k0" "$patch/stamp-revision.k0" "$stage/candidate.seme" \
    "$stage/digest" "$stage/final.seme"
"$k0" "$kernel" "$stage/final.seme"
"$k0" "$foundation" "$stage/final.seme"

mv -f "$stage/final.seme" "$output"
echo "Patch Module v1 transaction committed: $output"
