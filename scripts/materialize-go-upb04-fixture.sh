#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
destination=${1:?destination required}
if test -e "$destination"; then
  echo "destination already exists" >&2
  exit 1
fi
mkdir -m 700 "$destination"
complete=false
trap 'if test "$complete" != true; then rm -rf "$destination"; fi' EXIT HUP INT TERM
mkdir -m 700 "$destination/application" "$destination/model" "$destination/policy"
while read -r digest relative; do
  source=$repo/fixtures/go-uab-11/$relative
  actual=$(sha256sum "$source" | cut -d ' ' -f 1)
  test "$actual" = "$digest" || { echo "source drift: $relative" >&2; exit 1; }
  cp "$source" "$destination/$relative"
done < "$repo/fixtures/go-upb04-uab11-overlay/SOURCES.sha256"
cp "$repo/fixtures/go-upb04-uab11-overlay/go.mod" "$destination/go.mod"
cp "$repo/fixtures/go-upb04-uab11-overlay/go.sum" "$destination/go.sum"
complete=true
