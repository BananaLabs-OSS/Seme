#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-uab12-cross.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache"; XDG_CACHE_HOME="$work/cache"; export GOCACHE XDG_CACHE_HOME
module="$repo/modules/execution/v35/module.g1"
package=seme.uab11/application
cd "$repo"

# Lift the one declared shared application exactly once.
node reference/js/javascript-package-provider-cli.mjs \
  --files fixtures/javascript-uab-11/application.js,fixtures/javascript-uab-11/policy.js \
  --module "$module" --package "$package" --revision 1 --entry Apply --out "$work/seed.g1"
bootstrap/seme-k0-linux-amd64 compiler/g1-compiler.k0 "$work/seed.g1" "$work/seed.seme"
bootstrap/seme-k0-linux-amd64 compiler/kernel-wire-validator.k0 "$work/seed.seme"
node reference/js/javascript-uab-11-canonical-corpus.mjs \
  "$repo/fixtures/javascript-uab-11/application.js" "$work/requests.jsonl" "$work/expected.jsonl"
test "$(wc -l < "$work/expected.jsonl" | tr -d ' ')" -eq 2048
node reference/js/cross-language-uab12-native-summary.mjs "$work/expected.jsonl" > "$work/expected-summary.json"

# JavaScript opens the graph as ordinary native JavaScript and independently
# re-lifts the complete graph, including all semantic identities.
node reference/js/javascript-projector-cli.mjs "$work/seed.g1" "$work/projected.mjs"
node --check "$work/projected.mjs"
node reference/js/javascript-provider-cli.mjs --source "$work/projected.mjs" \
  --module "$module" --package "$package" --revision 1 --entry Apply --out "$work/js-relift.g1"
cmp "$work/seed.g1" "$work/js-relift.g1"
node reference/js/javascript-uab-11-canonical-corpus.mjs \
  "$work/projected.mjs" "$work/js-requests.jsonl" "$work/js-native.jsonl"
cmp "$work/requests.jsonl" "$work/js-requests.jsonl"
node reference/js/javascript-uab-11-jsonl-equal.mjs "$work/expected.jsonl" "$work/js-native.jsonl"

# Go opens the same graph. Its evidence adapter consumes/emits only canonical
# JSON values and has no application behavior of its own.
(cd reference/go &&
  go build -buildvcs=false -o "$work/go-projector" ./cmd/go-projector &&
  go build -buildvcs=false -o "$work/go-session" ./cmd/go-session-proof &&
  go build -buildvcs=false -o "$work/canonical-observe" ./cmd/canonical-observe)
mkdir -p "$work/go/application" "$work/go/cmd/evidence"
"$work/go-projector" -package application "$work/seed.g1" "$work/go/application/application.go"
cp targets/uab-v1/cross-language-go.mod "$work/go/go.mod"
cp targets/uab-v1/cross-language-go-runner/main.go "$work/go/cmd/evidence/main.go"
(cd "$work/go" && go test -buildvcs=false ./...)
"$work/go-session" --module "$module" --project "$work/go" --package "$package" \
  --entry Apply --revision 1 --out "$work/go-relift.g1"
cmp "$work/seed.g1" "$work/go-relift.g1"
(cd "$work/go" && go run -buildvcs=false ./cmd/evidence < "$work/requests.jsonl" > "$work/go-native.jsonl")
node reference/js/javascript-uab-11-jsonl-equal.mjs "$work/expected.jsonl" "$work/go-native.jsonl"

# Lua opens the same graph directly; no Go or JavaScript source participates in
# either its native execution or re-import.
node reference/lua/lua-projector-cli.mjs "$work/seed.g1" "$work/projected.lua"
node reference/lua/lua-provider-cli.mjs --source "$work/projected.lua" \
  --module "$module" --package "$package" --revision 1 --entry Apply --out "$work/lua-relift.g1"
cmp "$work/seed.g1" "$work/lua-relift.g1"
node reference/lua/lua-uab-11-corpus.mjs > "$work/lua-corpus.json"
: > "$work/no-policy.lua"
env XDG_DATA_HOME="$work/lua-data" XDG_STATE_HOME="$work/lua-state" XDG_CACHE_HOME="$work/lua-cache" \
  nvim -l reference/lua/lua-uab-11-native.lua reference/lua/seme-values.lua \
  "$work/no-policy.lua" "$work/projected.lua" "$work/lua-corpus.json" > "$work/lua-native.json"
node reference/js/json-equal.mjs "$work/expected-summary.json" "$work/lua-native.json"

# The canonical interpreter observes the same complete corpus. The reusable
# UAB-11 target gate then executes this deterministic seed through standalone
# Wasm and pinned Pulp, including atomic capability and malformed-input denial.
"$work/canonical-observe" "$work/seed.seme" < "$work/requests.jsonl" > "$work/canonical.jsonl"
node reference/js/javascript-uab-11-jsonl-equal.mjs "$work/expected.jsonl" "$work/canonical.jsonl"
"$repo/scripts/check-javascript-uab-11-target.sh"

# Nearby native coercive indexing is not assigned checked Core meaning in any
# language. Exact graph identity cannot substitute for negative evidence.
if node reference/js/javascript-provider-cli.mjs --source fixtures/javascript-uab-11/reject-raw-index.js \
  --module "$module" --package seme.uab12/reject-js --revision 1 --entry RawIndex --out "$work/reject-js.g1" \
  > "$work/reject-js.log" 2>&1; then exit 1; fi
rg -q 'javascript.raw_index_requires_adapter:' "$work/reject-js.log"
cp -R "$work/go" "$work/go-edited"
sed -i '0,/Error: 1/s//Error: 9/' "$work/go-edited/application/application.go"
if "$work/go-session" --module "$module" --project "$work/go-edited" --package "$package" \
  --entry Apply --revision 1 --out "$work/reject-go.g1" > "$work/reject-go.log" 2>&1; then exit 1; fi
rg -q 'go_projection.envelope_source_mismatch' "$work/reject-go.log"
test ! -e "$work/reject-go.g1"
printf '%s\n' '---@param values seme.slice<seme.i64>' '---@return seme.i64' \
  'function Raw(values)' '  return values[1]' 'end' > "$work/reject.lua"
if node reference/lua/lua-provider-cli.mjs --source "$work/reject.lua" --module "$module" \
  --package seme.uab12/reject-lua --revision 1 --entry Raw --out "$work/reject-lua.g1" \
  > "$work/reject-lua.log" 2>&1; then exit 1; fi
rg -q 'lua.unsupported_expression:.*reject.lua:4:1' "$work/reject-lua.log"

echo 'UAB-12 complete acceptance: one canonical application projects to Go, JavaScript, and Lua; all re-lifts are byte-identical and all five evidence classes pass'
