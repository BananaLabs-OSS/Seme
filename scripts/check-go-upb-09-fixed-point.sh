#!/bin/sh
# Candidate Project-v12 source-free projection and re-lift fixed-point gate.
# This script remains unclaimed until its serialized heavyweight run is green.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb09-fixed.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
proxy="$repo/fixtures/go-upb03-offline-proxy"
source="$work/source"; bundle="$work/bundle"; projected="$work/projected"; rebuilt="$work/rebuilt"
"$repo/scripts/materialize-go-upb09-fixture.sh" "$source"
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/build" ./cmd/go-upb09-build && go build -buildvcs=false -o "$work/project" ./cmd/go-upb09-project)
go_tool=$(cd "$repo/reference/go" && go env GOROOT)/bin/go
go_root=$(CDPATH= cd -- "$(dirname -- "$go_tool")/.." && pwd)
test -x "$go_tool"
test "$(env GOROOT="$go_root" "$go_tool" version)" = "go version go1.26.0 linux/amd64"

contracts="-execution-g1 $repo/modules/execution/v36/module.g1 -execution-contract $repo/modules/execution/v36/module.seme -foundation-contract $repo/modules/foundation/v1/module.seme -package-v4 $repo/modules/package/v4/module.seme -project-v8 $repo/modules/project/v8/module.seme -dependency-v1 $repo/modules/dependency/v1/module.seme -configuration-v3 $repo/modules/configuration/v3/module.seme -resource-v1 $repo/modules/resource/v1/module.seme -project-v9 $repo/modules/project/v9/module.seme -durable-state-v1 $repo/modules/durable-state/v1/module.seme -source-presentation-v1 $repo/modules/source-presentation/v1/module.seme -project-v10 $repo/modules/project/v10/module.seme -ordered-transport-v1 $repo/modules/ordered-transport/v1/module.seme -project-v11 $repo/modules/project/v11/module.seme -controlled-effects-v1 $repo/modules/controlled-effects/v1/module.seme -project-v12 $repo/modules/project/v12/module.seme -k0 $repo/bootstrap/seme-k0-linux-amd64 -g1-compiler $repo/compiler/g1-compiler.k0"
common="-proxy $proxy -module example.test/go-uab-11 -package example.test/go-uab-11/streamservice -entry DispatchControlled -revision 1 -dependency example.test/seme/checksum -version v1.2.3 -local-from example.test/go-uab-11/application -local-to example.test/go-uab-11/model -resource-owner example.test/go-uab-11/service"
project_inputs="-module example.test/go-uab-11 -go $go_tool -selection $source/configuration-selection.json -durable-selection $source/durable-selection.json -transport-selection $repo/fixtures/go-upb08-transport-overlay/transport-selection.json -effects-selection $source/controlled-effects-selection.json -foundation $repo/modules/foundation/v1/module.seme -execution $repo/modules/execution/v36/module.seme -package $repo/modules/package/v4/module.seme -dependency $repo/modules/dependency/v1/module.seme -configuration $repo/modules/configuration/v3/module.seme -resource $repo/modules/resource/v1/module.seme -durable-state $repo/modules/durable-state/v1/module.seme -source-presentation $repo/modules/source-presentation/v1/module.seme -project-v8 $repo/modules/project/v8/module.seme -project-v9 $repo/modules/project/v9/module.seme -project-v10 $repo/modules/project/v10/module.seme -ordered-transport $repo/modules/ordered-transport/v1/module.seme -project-v11 $repo/modules/project/v11/module.seme -controlled-effects $repo/modules/controlled-effects/v1/module.seme -project-v12 $repo/modules/project/v12/module.seme -k0 $repo/bootstrap/seme-k0-linux-amd64 -g1-compiler $repo/compiler/g1-compiler.k0"
# shellcheck disable=SC2086
"$work/build" -project "$source" -out "$bundle" $common $contracts -selection "$source/configuration-selection.json" -resources "$source/resources.json" -durable-selection "$source/durable-selection.json" -transport-selection "$repo/fixtures/go-upb08-transport-overlay/transport-selection.json" -effects-selection "$source/controlled-effects-selection.json"

# shellcheck disable=SC2086
"$work/project" -bundle "$bundle" -to "$projected" $project_inputs

test -z "$(find "$projected" -type f \( -name '*.seme' -o -name '*.g1' \) -print -quit)"
cmp "$source/controlled-effects-selection.json" "$projected/controlled-effects-selection.json"
cmp "$repo/fixtures/go-upb08-transport-overlay/transport-selection.json" "$projected/ordered-transport-selection.json"
cmp "$source/resources/notice.txt" "$projected/resources/notice.txt"
cmp "$source/resources/marker.bin" "$projected/resources/marker.bin"
(cd "$projected" && GOROOT="$go_root" GOPROXY=off GOSUMDB=off "$go_tool" test -count=1 ./...)

# shellcheck disable=SC2086
"$work/build" -project "$projected" -out "$rebuilt" $common $contracts -selection "$projected/configuration-selection.json" -resources "$projected/resources.json" -durable-selection "$projected/durable-selection.json" -transport-selection "$projected/ordered-transport-selection.json" -effects-selection "$projected/controlled-effects-selection.json"
for artifact in construction-v36.g1 controlled-effects-v1.seme project-v12.seme ordered-transport-v1.seme project-v11.seme; do cmp "$bundle/$artifact" "$rebuilt/$artifact"; done

mkdir "$work/existing"
# shellcheck disable=SC2086
if "$work/project" -bundle "$bundle" -to "$work/existing" $project_inputs 2>/dev/null; then exit 1; fi
cp -R "$bundle" "$work/tampered"
printf x >> "$work/tampered/project-v12.seme"
# shellcheck disable=SC2086
if "$work/project" -bundle "$work/tampered" -to "$work/rejected" $project_inputs 2>/dev/null; then exit 1; fi
test ! -e "$work/rejected"
printf 'Go UPB-09 Project-v12 source-free projection/re-lift fixed point passes\n'
