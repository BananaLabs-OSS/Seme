#!/bin/sh
# Independently project one authenticated canonical seed into three native trees.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
authority=${1:?authority required};module=${2:?canonical module required};destination=${3:?new destination required}
test ! -e "$destination";parent=$(dirname "$destination");test "$(cd "$parent"&&pwd -P)" = "$parent"
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-upb12-project.XXXXXX");trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache";export GOCACHE
(cd "$repo/reference/go"&&go build -buildvcs=false -o "$work/graph" ./cmd/upb12-projection-graph&&go build -buildvcs=false -o "$work/go-project" ./cmd/go-project-v4-projector)
contracts="-foundation $repo/modules/foundation/v1/module.seme -execution $repo/modules/execution/v36/module.seme -packages $repo/modules/package/v4/module.seme -dependency $repo/modules/dependency/v1/module.seme -configuration $repo/modules/configuration/v3/module.seme -project $repo/modules/project/v8/module.seme"
# shellcheck disable=SC2086 -- the fixed repository-owned contract argument list is intentional.
"$work/graph" -authority "$authority" -language javascript -module "$module" $contracts -out "$work/javascript-graph.json"
# shellcheck disable=SC2086
"$work/graph" -authority "$authority" -language lua -module "$module" $contracts -out "$work/lua-graph.json"
stage=$(mktemp -d "$parent/.seme-upb12-native.XXXXXX");published=false
cleanup(){ if test "$published" = false;then rm -rf "$stage";fi;rm -rf "$work";};trap cleanup EXIT HUP INT TERM
# shellcheck disable=SC2086
"$work/go-project" -authority "$authority" -module "$module" $contracts -out "$stage/go"
node "$repo/reference/js/javascript-module-projector-cli.mjs" "$authority/construction-v36.g1" "$work/javascript-graph.json" "$stage/javascript"
node "$repo/reference/lua/lua-module-projector-cli.mjs" "$authority/construction-v36.g1" "$work/lua-graph.json" "$stage/lua"
cp "$work/javascript-graph.json" "$stage/javascript/package-graph.json"
cp "$work/lua-graph.json" "$stage/lua/package-graph.json"
cp "$repo/reference/lua/seme-values.lua" "$stage/lua/seme-values.lua"
mkdir -p "$stage/go/controlled"
cp "$repo/reference/upb12/native/go-controlled-test.go" "$stage/go/controlled/upb12_native_test.go"
cp "$repo/reference/upb12/native/javascript-native-test.mjs" "$stage/javascript/native-test.mjs"
cp "$repo/reference/upb12/native/javascript-package.json" "$stage/javascript/package.json"
cp "$repo/reference/upb12/native/lua-native-test.lua" "$stage/lua/native-test.lua"
mv -T -n "$stage" "$destination";test ! -e "$stage";published=true
printf 'UPB12 projected one authority into Go, JavaScript, and Lua native module trees\n'
