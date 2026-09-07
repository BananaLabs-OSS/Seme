#!/bin/sh
set -eu

if [ "$#" -ne 5 ]; then
    echo "usage: build-certified-canonical-v1.sh INPUT.seme EXPECTED_REVISION EXPECTED_SHA256 OUTPUT.wasm OUTPUT.json" >&2
    exit 64
fi

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
input=$1
expected_revision=$2
expected_sha256=$3
output=$4
evidence=$5

case "$expected_revision" in
    *[!0123456789abcdef]*|'') valid_revision=0 ;;
    *) valid_revision=1 ;;
esac
if [ "$valid_revision" -ne 1 ] || [ "${#expected_revision}" -ne 32 ]; then
    echo "canonical-build: expected revision must be 32 lowercase hexadecimal characters" >&2
    exit 64
fi
case "$expected_sha256" in
    *[!0123456789abcdef]*|'') valid_sha256=0 ;;
    *) valid_sha256=1 ;;
esac
if [ "$valid_sha256" -ne 1 ] || [ "${#expected_sha256}" -ne 64 ]; then
    echo "canonical-build: expected SHA-256 must be 64 lowercase hexadecimal characters" >&2
    exit 64
fi

[ -f "$input" ] || { echo "canonical-build: input does not exist: $input" >&2; exit 66; }
for tool in go sha256sum awk grep cmp mktemp; do
    command -v "$tool" >/dev/null 2>&1 || {
        echo "canonical-build: missing required tool: $tool" >&2
        exit 69
    }
done

input_abs=$(CDPATH= cd -- "$(dirname -- "$input")" && printf '%s/%s\n' "$PWD" "$(basename -- "$input")")
output_dir=$(CDPATH= cd -- "$(dirname -- "$output")" && pwd) || {
    echo "canonical-build: output directory does not exist: $(dirname -- "$output")" >&2
    exit 73
}
evidence_dir=$(CDPATH= cd -- "$(dirname -- "$evidence")" && pwd) || {
    echo "canonical-build: evidence directory does not exist: $(dirname -- "$evidence")" >&2
    exit 73
}
output_abs=$output_dir/$(basename -- "$output")
evidence_abs=$evidence_dir/$(basename -- "$evidence")
if [ "$input_abs" = "$output_abs" ] || [ "$input_abs" = "$evidence_abs" ] || \
    [ "$output_abs" = "$evidence_abs" ]; then
    echo "canonical-build: input, artifact, and evidence paths must be distinct" >&2
    exit 64
fi

actual_sha256=$(sha256sum "$input_abs" | awk '{print $1}')
if [ "$actual_sha256" != "$expected_sha256" ]; then
    echo "canonical-build: canonical digest mismatch: expected $expected_sha256, got $actual_sha256" >&2
    exit 65
fi

work=$(mktemp -d "${TMPDIR:-/tmp}/seme-canonical-build.XXXXXX")
wasm_stage=
evidence_stage=
cleanup() {
    rm -rf "$work"
    [ -z "$wasm_stage" ] || rm -f "$wasm_stage"
    [ -z "$evidence_stage" ] || rm -f "$evidence_stage"
}
stop() {
    status=$1
    trap - EXIT HUP INT TERM
    cleanup
    exit "$status"
}
trap cleanup EXIT
trap 'stop 129' HUP
trap 'stop 130' INT
trap 'stop 143' TERM

k0=$repo/bootstrap/seme-k0-linux-amd64
"$k0" "$repo/compiler/kernel-wire-validator.k0" "$input_abs"
"$k0" "$repo/modules/foundation/v1/validator.k0" "$input_abs" \
    "$work/validated.seme"
cmp "$input_abs" "$work/validated.seme"

(cd "$repo/reference/go" && \
    go build -buildvcs=false -o "$work/pure-wasm-lower" ./cmd/pure-wasm-lower)
"$work/pure-wasm-lower" "$input_abs" "$work/artifact.wasm" "$work/evidence.json"

abi_revision=$(awk -F '"' '$2 == "canonical_revision" { print $4; exit }' "$work/evidence.json")
abi_program_sha256=$(awk -F '"' '$2 == "program_sha256" { print $4; exit }' "$work/evidence.json")
abi_artifact_sha256=$(awk -F '"' '$2 == "artifact_sha256" { print $4; exit }' "$work/evidence.json")
actual_artifact_sha256=$(sha256sum "$work/artifact.wasm" | awk '{print $1}')

[ "$abi_revision" = "$expected_revision" ] || {
    echo "canonical-build: canonical revision mismatch: expected $expected_revision, got ${abi_revision:-missing}" >&2
    exit 65
}
[ "$abi_program_sha256" = "$actual_sha256" ] || {
    echo "canonical-build: lowerer evidence does not identify the supplied canonical bytes" >&2
    exit 65
}
[ "$abi_artifact_sha256" = "$actual_artifact_sha256" ] || {
    echo "canonical-build: artifact digest does not match lowerer evidence" >&2
    exit 65
}

wasm_stage=$(mktemp "$output_dir/.seme-canonical-wasm.XXXXXX")
evidence_stage=$(mktemp "$evidence_dir/.seme-canonical-evidence.XXXXXX")
cp "$work/artifact.wasm" "$wasm_stage"
cp "$work/evidence.json" "$evidence_stage"
chmod 0644 "$wasm_stage" "$evidence_stage"
mv -f "$wasm_stage" "$output_abs"
wasm_stage=
mv -f "$evidence_stage" "$evidence_abs"
evidence_stage=

echo "canonical-build: certified revision $expected_revision"
echo "canonical-build: artifact $output_abs ($actual_artifact_sha256)"
echo "canonical-build: evidence $evidence_abs"
