#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-go-upb-01.sh"
"$repo/scripts/check-go-upb-02.sh"
"$repo/scripts/check-go-upb-03.sh"
"$repo/scripts/check-upb-v1-scorecard.sh"
echo 'UPB-v1 claimed cells: every mapped evidence gate passed'
