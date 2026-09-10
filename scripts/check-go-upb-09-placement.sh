#!/bin/sh
# Authenticated UPB09 placement evidence. Pulp carries only the capability-free
# pure planner/replay; injected clock and effect delivery remain a Go host port.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
manifest="$repo/targets/wasm/pulp-function-composite-v1/pulp.cell.toml"
digest=e88c5c9fb7d4c839090dcbc546d838466d6a68b8d818ce3fa9e8da7efeea826c
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb09-placement.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/cache"; export GOCACHE
test -f "$pulp_repo/go.mod"
test "$(git -C "$pulp_repo" rev-parse "$commit^{commit}")" = "$commit"
test "$(sha256sum "$manifest" | cut -d ' ' -f 1)" = "$digest"
mkdir "$work/pinned"
git -C "$pulp_repo" archive "$commit" | tar -x -C "$work/pinned"
node - "$manifest" <<'NODE'
const text=require('fs').readFileSync(process.argv[2],'utf8');
const one=(r,n)=>{const x=[...text.matchAll(r)];if(x.length!==1)throw Error(n);return x[0][1].trim()};
if(one(/^provides\s*=\s*\[(.*)\]\s*$/gm,'provides')!=='"seme.function-composite-v1"')throw Error('provider');
if(one(/^consumes\s*=\s*\[(.*)\]\s*$/gm,'consumes')!==''||one(/^capabilities\s*=\s*\[(.*)\]\s*$/gm,'capabilities')!=='')throw Error('ambient authority');
NODE
cat > "$work/pinned/ext/upb09_placement_test.go" <<'EOF'
package ext
import "testing"
func TestUPB09AmbientProvidersCannotSubstitute(t *testing.T){registered:=[]Capability{{Name:"http",Provider:"pulp.http"},{Name:"websocket",Provider:"pulp.websocket"},{Name:"storage.fs",Provider:"github.com/BananaLabs-OSS/Pulp-ext-fs"}};for _,name:=range []string{"clock.injected.unix-milliseconds.v1","random.seeded.lcg-48271-plus-1.v1","observability.log","wasi.clock","entropy.read"}{if got,e:=SelectCapabilities(registered,map[string]string{name:"seme.controlled-effects.v1"});e==nil{t.Fatalf("false provider %s: %#v",name,got)}}}
EOF
(cd "$work/pinned" && go test -count=1 ./ext -run '^TestUPB09AmbientProvidersCannotSubstitute$')
(cd "$repo/reference/go" && go test -count=1 ./controlledeffectsplacement)
echo 'Go UPB-09 placement: Go host clock/effect boundary authenticated; pure canonical/Wasm/Pulp placement exact; ambient substitutions rejected'
