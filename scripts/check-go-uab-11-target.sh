#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
UAB11_LANGUAGE=go; export UAB11_LANGUAGE
exec "$repo/scripts/check-javascript-uab-11-target.sh"
