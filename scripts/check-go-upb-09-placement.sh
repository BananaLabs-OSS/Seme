#!/bin/sh
# Authenticated UPB09 placement evidence. Pulp carries only the capability-free
# pure planner/replay; injected clock and effect delivery remain a Go host port.
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}
commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
manifest="$repo/targets/wasm/pulp-function-composite-v1/pulp.cell.toml"
digest=e88c5c9fb7d4c839090dcbc546d838466d6a68b8d818ce3fa9e8da7efeea826c
observed_manifest="$repo/targets/wasm/pulp-canonical-vm-v1/pulp.cell.toml"
observed_digest=12f65a8f865a068b2c48c0b749a4f48ea3519752ccb1db1be9abb9e5c445167d
observed_runner="$repo/targets/wasm/pulp-canonical-vm-v1/runner.go"
runner_digest=01ef0baa21f882720829cb63d695538e3bfe19f7d256ecea5299085f9ee428d9
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-upb09-placement.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/cache"; export GOCACHE
test -f "$pulp_repo/go.mod"
test "$(git -C "$pulp_repo" rev-parse "$commit^{commit}")" = "$commit"
test "$(sha256sum "$manifest" | cut -d ' ' -f 1)" = "$digest"
test "$(sha256sum "$observed_manifest" | cut -d ' ' -f 1)" = "$observed_digest"
test "$(sha256sum "$observed_runner" | cut -d ' ' -f 1)" = "$runner_digest"
mkdir "$work/pinned"
git -C "$pulp_repo" archive "$commit" | tar -x -C "$work/pinned"
node - "$manifest" <<'NODE'
const text=require('fs').readFileSync(process.argv[2],'utf8');
const one=(r,n)=>{const x=[...text.matchAll(r)];if(x.length!==1)throw Error(n);return x[0][1].trim()};
if(one(/^provides\s*=\s*\[(.*)\]\s*$/gm,'provides')!=='"seme.function-composite-v1"')throw Error('provider');
if(one(/^consumes\s*=\s*\[(.*)\]\s*$/gm,'consumes')!==''||one(/^capabilities\s*=\s*\[(.*)\]\s*$/gm,'capabilities')!=='')throw Error('ambient authority');
NODE
# The runtime parity harness is a second, distinct placement. Its single log
# capability captures the logical request for comparison; it is not evidence
# that production external delivery occurred.
node - "$observed_manifest" <<'NODE'
const text=require('fs').readFileSync(process.argv[2],'utf8');
const one=(r,n)=>{const x=[...text.matchAll(r)];if(x.length!==1)throw Error(n);return x[0][1].trim()};
if(one(/^provides\s*=\s*\[(.*)\]\s*$/gm,'provides')!=='"seme.evaluate.v1"')throw Error('provider');
if(one(/^consumes\s*=\s*\[(.*)\]\s*$/gm,'consumes')!=='')throw Error('consume');
if(one(/^capabilities\s*=\s*\[(.*)\]\s*$/gm,'capabilities')!=='"observability.log"')throw Error('capability');
NODE
rg -q 'Name: "observability.log", Provider: "seme.pulp.log-v1"' "$observed_runner"
rg -q 'cell.Call\(ctx, "seme.evaluate.v1", request\)' "$observed_runner"
if rg -q 'clock_time_get|entropy.read|random_get' "$observed_runner"; then
  echo 'observed parity runner contains ambient clock or entropy substitution' >&2
  exit 1
fi
cat > "$work/pinned/ext/upb09_placement_test.go" <<'EOF'
package ext
import "testing"
func TestUPB09AmbientProvidersCannotSubstitute(t *testing.T){registered:=[]Capability{{Name:"http",Provider:"pulp.http"},{Name:"websocket",Provider:"pulp.websocket"},{Name:"storage.fs",Provider:"github.com/BananaLabs-OSS/Pulp-ext-fs"}};for _,name:=range []string{"clock.injected.unix-milliseconds.v1","random.seeded.lcg-48271-plus-1.v1","observability.log","wasi.clock","entropy.read"}{if got,e:=SelectCapabilities(registered,map[string]string{name:"seme.controlled-effects.v1"});e==nil{t.Fatalf("false provider %s: %#v",name,got)}}}
EOF
(cd "$work/pinned" && go test -count=1 ./ext -run '^TestUPB09AmbientProvidersCannotSubstitute$')
(cd "$repo/reference/go" && go test -count=1 ./controlledeffectsplacement)
echo 'Go UPB-09 placement: Go host delivery, capability-free pure carrier, and observed canonical-VM parity harness remain distinct; ambient substitutions rejected'
