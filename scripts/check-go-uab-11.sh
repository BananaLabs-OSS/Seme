#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
"$repo/scripts/check-go-uab-11-source.sh"
"$repo/scripts/check-go-uab-11-target.sh"
echo 'Go UAB-11 complete acceptance: all five evidence classes pass'
