#!/bin/sh
# Apply one native edit in all three ecosystems and prove that each exact
# source-bound re-lift recovers the same independently rebuilt Seme revision.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
prior=${1:?prior source-free authority required}; result=${2:?result source-free authority required}
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-upb12-convergence.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
target=807557c5737cee0105a083f3608c7d07
contracts="-foundation $repo/modules/foundation/v1/module.seme -execution $repo/modules/execution/v36/module.seme -packages $repo/modules/package/v4/module.seme -dependency $repo/modules/dependency/v1/module.seme -configuration $repo/modules/configuration/v3/module.seme -project $repo/modules/project/v8/module.seme"
"$repo/scripts/project-upb12-authority.sh" "$prior" seme.upb12/service "$work/prior"
"$repo/scripts/project-upb12-authority.sh" "$result" seme.upb12/service "$work/result"
for language in go javascript lua; do graph="$work/prior/$language/package-graph.json"; test "$language" = go && graph="$work/prior/javascript/package-graph.json"; node "$repo/reference/js/upb12-native-edit-cli.mjs" --project "$work/prior/$language" --graph "$graph" --language "$language" --target "$target" --expected InitializePolicy --replacement BuildPolicy --out "$work/edited-$language"; done
cmp "$work/edited-go/SEME-EDIT.json" "$work/edited-javascript/SEME-EDIT.json"; cmp "$work/edited-go/SEME-EDIT.json" "$work/edited-lua/SEME-EDIT.json"
(cd "$work/edited-go" && go test -count=1 ./...)
(cd "$work/edited-javascript" && npm test)
env XDG_DATA_HOME="$work/lua-data" XDG_STATE_HOME="$work/lua-state" XDG_CACHE_HOME="$work/lua-cache" nvim --clean -u NONE -n --headless -l "$work/edited-lua/native-test.lua" "$work/edited-lua"
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/go-relift" ./cmd/upb12-go-relift && go build -buildvcs=false -o "$work/revision-verify" ./cmd/upb12-revision-verify)
# shellcheck disable=SC2086 -- fixed repository-owned contract flags.
"$work/go-relift" -authority "$result" -project-root "$work/edited-go" -module seme.upb12/service $contracts -out "$work/go.g1"
node "$repo/reference/js/upb12-native-relift-cli.mjs" --authority "$result" --graph "$work/result/javascript/package-graph.json" --project "$work/edited-javascript" --language javascript --out "$work/javascript.g1"
node "$repo/reference/js/upb12-native-relift-cli.mjs" --authority "$result" --graph "$work/result/lua/package-graph.json" --project "$work/edited-lua" --language lua --out "$work/lua.g1"
for language in go javascript lua; do cmp "$result/construction-v36.g1" "$work/$language.g1"; done
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$prior/construction-v36.g1" "$work/prior.seme"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$result/construction-v36.g1" "$work/result.seme"
"$work/revision-verify" -prior "$work/prior.seme" -result "$work/result.seme" -target "$target" -expected InitializePolicy -replacement BuildPolicy
printf 'UPB12 revision convergence: Go, JavaScript, and Lua native edits recover one exact canonical revision\n'
