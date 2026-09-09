#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd); pulp_repo=${PULP_REPO:-"$repo/../Pulp"}; commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-go-uab02-aggregate.XXXXXX"); trap 'rm -rf "$work"' EXIT HUP INT TERM
XDG_CACHE_HOME="$work/cache"; GOCACHE="$work/go-build"; export XDG_CACHE_HOME GOCACHE
(cd "$repo/reference/go" && go test -count=1 -buildvcs=false ./goprovider ./goprojector ./canonicaleval ./wasmtarget && go build -buildvcs=false -o "$work/session" ./cmd/go-session-proof && go build -buildvcs=false -o "$work/projector" ./cmd/go-projector && go build -buildvcs=false -o "$work/lower" ./cmd/aggregate-wasm-lower && go build -buildvcs=false -o "$work/eval" ./cmd/canonical-eval)
(cd "$repo/fixtures/go-uab-02-aggregate" && go run -buildvcs=false ./cmd/native-observations > "$work/native.json")
"$work/session" --module "$repo/modules/execution/v32/module.g1" --project "$repo/fixtures/go-uab-02-aggregate" --package example.test/go-uab-02-aggregate --entry Observe --revision 1 --out "$work/source.g1"
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/source.g1" "$work/source.seme"
"$work/eval" "$work/source.seme" "$repo/fixtures/go-uab-02-aggregate/canonical-vectors.json" > "$work/canonical.json"
rg -q '"i64":"26"' "$work/canonical.json"; rg -q '"i64":"15"' "$work/canonical.json"; rg -q '"i64":"-9223372036854775808"' "$work/canonical.json"; rg -q '"malformed":2' "$work/canonical.json"
"$work/projector" -package aggregate "$work/source.g1" "$work/projected.go"
mkdir -p "$work/projected/cmd/native-observations"; cp "$repo/fixtures/go-uab-02-aggregate/go.mod" "$work/projected/go.mod"; cp "$work/projected.go" "$work/projected/program.go"; cp "$repo/fixtures/go-uab-02-aggregate/cmd/native-observations/main.go" "$work/projected/cmd/native-observations/main.go"
(cd "$work/projected" && go run -buildvcs=false ./cmd/native-observations > "$work/projected.json"); cmp "$work/native.json" "$work/projected.json"
"$work/session" --module "$repo/modules/execution/v32/module.g1" --project "$work/projected" --package example.test/go-uab-02-aggregate --entry Observe --revision 1 --out "$work/relift.g1"; cmp "$work/source.g1" "$work/relift.g1"
"$work/lower" "$work/source.seme" "$work/program.wasm" "$work/abi.json"
node "$repo/reference/js/aggregate-function-runner.mjs" "$work/program.wasm" "$repo/fixtures/go-uab-02-aggregate/vectors.json" > "$work/wasm.json"
rg -q '"present":"26"' "$work/wasm.json"; rg -q '"missing":"15"' "$work/wasm.json"; rg -q '"wrapping":"-9223372036854775808"' "$work/wasm.json"
node "$repo/reference/js/aggregate-function-runner.mjs" "$work/program.wasm" "$repo/fixtures/go-uab-02-aggregate/vectors.json" --tsv > "$work/vectors.tsv"
mkdir "$work/pulp" "$work/pinned"; git -C "$pulp_repo" archive "$commit" | tar -x -C "$work/pinned"; (cd "$work/pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof); cp "$repo/targets/wasm/pulp-function-aggregate-v1/pulp.cell.toml" "$work/pulp/pulp.cell.toml"; cp "$work/program.wasm" "$work/pulp/pure-function.wasm"
while IFS="$(printf '\t')" read -r kind name request response; do if [ "$kind" = valid ]; then "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-aggregate-v1 -request "$request" > "$work/$name.log" 2>&1; rg -q "\"response\":\"$response\"" "$work/$name.log"; else if "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-aggregate-v1 -request "$request" >/dev/null 2>&1; then echo "malformed aggregate accepted" >&2; exit 1; fi; fi; done < "$work/vectors.tsv"
echo "Go UAB-02 aggregate: shared record/array/slice/map observations passed native, canonical, projected, Wasm, and Pulp"
