#!/bin/sh
# Focused Lua identity-bound project reconciliation proof. Only the cumulative
# UPB-11 gate may claim the scorecard cell.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.."&&pwd);fixture="$repo/fixtures/lua-upb05-configuration";work=$(mktemp -d "${TMPDIR:-/tmp}/seme-lua-upb11-reconcile.XXXXXX");trap 'rm -rf "$work"' EXIT HUP INT TERM
files=application.lua,configuration.lua,controlled.lua,policy.lua,state.lua,transport.lua;structured=configuration-selection.json,durable-selection.json,transport-selection.json,controlled-effects-selection.json;target=80555555555555555555555555555506
revision(){ node --input-type=module - "$repo/reference/lua/lua-project-reconcile.mjs" "$1" <<'NODE'
const{luaProjectRevision:r}=await import(process.argv[2]);process.stdout.write(r({project:process.argv[3],files:["application.lua","configuration.lua","controlled.lua","policy.lua","state.lua","transport.lua"],structuredReferences:["configuration-selection.json","durable-selection.json","transport-selection.json","controlled-effects-selection.json"]}));
NODE
}
base=$(revision "$fixture")
run(){ node "$repo/reference/lua/lua-project-reconcile-cli.mjs" --project "$1" --out "$2" --project-path example.test/lua-upb05 --files "$files" --structured-references "$structured" --module "$repo/modules/execution/v36/module.g1" --entry Run --target "$3" --expected InitializePolicy --replacement "$4" --revision 2 --base-revision "$5" --native-runner "$repo/reference/lua/lua-upb11-native.lua" --native-adapter "$repo/reference/lua/seme-values.lua";}
verify(){ node "$repo/reference/lua/lua-project-reconcile-verify-cli.mjs" --project "$1" --project-path example.test/lua-upb05 --files "$files" --structured-references "$structured" --module "$repo/modules/execution/v36/module.g1" --entry Run --native-runner "$repo/reference/lua/lua-upb11-native.lua" --native-adapter "$repo/reference/lua/seme-values.lua";}
run "$fixture" "$work/a" "$target" BuildPolicy "$base";run "$fixture" "$work/b" "$target" BuildPolicy "$base";diff -ru "$work/a" "$work/b";verify "$work/a" >"$work/report.json"
rg -q 'local function BuildPolicy' "$work/a/policy.lua";rg -Fq 'return { BuildPolicy = BuildPolicy }' "$work/a/policy.lua";rg -q '"name":"BuildPolicy"' "$work/a/configuration-selection.json";rg -q 'InitializePolicy is historical prose' "$work/a/policy.lua";test -s "$work/a/.seme-reconciliation-v1/COMPLETE.sha256"
reject(){ name=$1;identity=$2;replacement=$3;if run "$fixture" "$work/reject-$name" "$identity" "$replacement" "$base" >"$work/$name.out" 2>"$work/$name.err";then echo "Lua UPB-11 accepted $name" >&2;exit 1;fi;test ! -e "$work/reject-$name";test ! -s "$work/$name.out";}
reject forged 00000000000000000000000000000000 BuildPolicy;reject collision "$target" Run;reject keyword "$target" function
cp -R "$fixture" "$work/stale";printf '\n-- concurrent native edit\n' >>"$work/stale/policy.lua";if run "$work/stale" "$work/reject-stale" "$target" BuildPolicy "$base" >"$work/stale.out" 2>"$work/stale.err";then echo 'Lua UPB-11 accepted simultaneous native and semantic changes' >&2;exit 1;fi;test ! -e "$work/reject-stale";test ! -s "$work/stale.out"
if run "$fixture" "$work/a" "$target" BuildPolicy "$base" >"$work/existing.out" 2>"$work/existing.err";then echo 'Lua UPB-11 overwrote an existing destination' >&2;exit 1;fi;test ! -s "$work/existing.out"
cp -R "$work/a" "$work/tampered";printf x >>"$work/tampered/.seme-reconciliation-v1/result-provider.g1";if verify "$work/tampered" >"$work/tampered.out" 2>"$work/tampered.err";then echo 'Lua UPB-11 accepted metadata tamper' >&2;exit 1;fi;test ! -s "$work/tampered.out"
node --test "$repo/reference/lua/lua-project-reconcile.test.mjs"
echo 'Lua UPB-11 focused reconciliation: stable annotated identity, declaration/export/config projection, native parity, deterministic publication, session monotonicity, and atomic adversaries pass'
