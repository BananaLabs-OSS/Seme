#!/bin/sh
# Apply one native edit in all three ecosystems and prove that each exact
# source-bound re-lift recovers the same independently rebuilt Seme revision.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
prior=${1:?prior source-free authority required}; result=${2:?result source-free authority required}
evidence_destination=${3:-}; test -z "$evidence_destination" || test ! -e "$evidence_destination"
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
if test -n "$evidence_destination"; then
  mkdir "$work/evidence" "$work/evidence/.seme-reconciliation-v1"
  cp "$prior/construction-v36.g1" "$work/evidence/.seme-reconciliation-v1/prior-provider.g1"
  cp "$result/construction-v36.g1" "$work/evidence/.seme-reconciliation-v1/result-provider.g1"
  result_digest=$(sha256sum "$result/construction-v36.g1" | cut -d ' ' -f 1)
  printf 'seme-upb12-shared-native-validation-v1\nlanguages go,javascript,lua\nobservations 4096\ncanonical_sha256 %s\n' "$result_digest" > "$work/evidence/.seme-reconciliation-v1/native-validation.txt"
  node - "$work/evidence/.seme-reconciliation-v1/projection-report.json" "$target" <<'NODE'
const fs=require("fs"),target=process.argv[3];fs.writeFileSync(process.argv[2],JSON.stringify({version:1,Language:"shared",Project:"seme.upb12/service",client_revision:2,Target:target,Field:"00000000000000000000000000009110",Expected:"InitializePolicy",Replacement:"BuildPolicy",Occurrences:[{language:"go"},{language:"javascript"},{language:"lua"}]})+"\n");
NODE
  node - "$work/evidence/.seme-reconciliation-v1" <<'NODE'
const fs=require("fs"),crypto=require("crypto"),root=process.argv[2],names=["native-validation.txt","prior-provider.g1","projection-report.json","result-provider.g1"].sort();let out="seme-upb12-shared-reconciliation-v1\n";for(const name of names)out+=`${name} ${crypto.createHash("sha256").update(fs.readFileSync(`${root}/${name}`)).digest("hex")}\n`;fs.writeFileSync(`${root}/COMPLETE.sha256`,out);
NODE
  evidence_parent=$(dirname "$evidence_destination"); test "$(cd "$evidence_parent" && pwd -P)" = "$evidence_parent"; mv -T -n "$work/evidence" "$evidence_destination"; test ! -e "$work/evidence"
fi
printf 'UPB12 revision convergence: Go, JavaScript, and Lua native edits recover one exact canonical revision\n'
