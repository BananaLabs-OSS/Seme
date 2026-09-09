#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-uab-v1-complete.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"
XDG_CACHE_HOME="$work/cache"
export GOCACHE XDG_CACHE_HOME
cd "$repo"

# UAB-01 through UAB-11 are independently owned by each language bridge.
# UAB-12 is one shared cross-language assertion and therefore runs once.
for gate in \
  scripts/check-uab-v1-go-01.sh \
  scripts/check-uab-v1-go-02.sh \
  scripts/check-go-uab-03.sh \
  scripts/check-go-uab-04.sh \
  scripts/check-go-uab-05.sh \
  scripts/check-go-uab-06.sh \
  scripts/check-go-uab-07.sh \
  scripts/check-go-uab-08.sh \
  scripts/check-go-uab-09.sh \
  scripts/check-go-uab-10.sh \
  scripts/check-go-uab-11.sh \
  scripts/check-javascript-uab-01.sh \
  scripts/check-javascript-uab-02.sh \
  scripts/check-javascript-uab-03.sh \
  scripts/check-javascript-uab-04.sh \
  scripts/check-javascript-uab-05.sh \
  scripts/check-javascript-uab-06.sh \
  scripts/check-javascript-uab-07.sh \
  scripts/check-javascript-uab-08.sh \
  scripts/check-javascript-uab-09.sh \
  scripts/check-javascript-uab-10.sh \
  scripts/check-javascript-uab-11.sh \
  scripts/check-lua-provider-v1.sh \
  scripts/check-lua-uab-02.sh \
  scripts/check-lua-uab-03.sh \
  scripts/check-lua-uab-04-native.sh \
  scripts/check-lua-uab-05.sh \
  scripts/check-lua-uab-06.sh \
  scripts/check-lua-uab-07.sh \
  scripts/check-lua-uab-08.sh \
  scripts/check-lua-uab-09.sh \
  scripts/check-lua-uab-10.sh \
  scripts/check-lua-uab-11.sh \
  scripts/check-uab-v1-cross-language-12.sh
do
  echo "UAB-v1 evidence: $gate"
  "$repo/$gate"
done

"$repo/scripts/check-uab-v1-scorecard.sh"
echo 'UAB-v1 complete profile: all 36 claimed cells passed their authoritative evidence gates'
