#!/bin/sh
set -eu
repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd); fixture="$repo/fixtures/javascript-upb04-typed"; vectors="$repo/fixtures/javascript-uab-04/vectors.json"
pulp_repo=${PULP_REPO:-"$repo/../Pulp"}; pulp_commit=acc66ca61fe69c5f2c4093bc55e13aeac6dcc001
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-javascript-upb04-parity.XXXXXX");trap 'rm -rf "$work"' EXIT HUP INT TERM
GOCACHE="$work/go-cache";XDG_CACHE_HOME="$work/cache";export GOCACHE XDG_CACHE_HOME
(cd "$fixture" && node "$repo/reference/js/javascript-package-provider-cli.mjs" --files application.js,policy/collections.js --module "$repo/modules/execution/v35/module.g1" --package example.test/javascript-upb04 --entry Run --revision 1 --out "$work/execution.g1")
"$repo/bootstrap/seme-k0-linux-amd64" "$repo/compiler/g1-compiler.k0" "$work/execution.g1" "$work/execution.seme"
node "$repo/reference/js/javascript-uab-04-native-runner.mjs" "$fixture/application.js" "$vectors" Run > "$work/native.json"
node "$repo/reference/js/javascript-uab-04-vectors.mjs" expected "$vectors" > "$work/expected.json"
node "$repo/reference/js/javascript-uab-04-compare.mjs" "$work/expected.json" "$work/native.json"
node "$repo/reference/js/javascript-uab-04-vectors.mjs" canonical "$vectors" > "$work/canonical-vectors.json"
(cd "$repo/reference/go" && go build -buildvcs=false -o "$work/eval" ./cmd/canonical-eval && go build -buildvcs=false -o "$work/lower" ./cmd/pure-wasm-lower)
"$work/eval" "$work/execution.seme" "$work/canonical-vectors.json" > "$work/canonical.json"
node "$repo/reference/js/javascript-uab-04-compare.mjs" "$work/expected.json" "$work/canonical.json"
"$work/lower" "$work/execution.seme" "$work/program.wasm" "$work/abi.json"
node "$repo/reference/js/javascript-uab-04-wasm-runner.mjs" "$work/program.wasm" "$vectors" > "$work/wasm.json"
node "$repo/reference/js/javascript-uab-04-compare.mjs" "$work/expected.json" "$work/wasm.json"
node "$repo/reference/js/javascript-uab-04-wasm-runner.mjs" "$work/program.wasm" "$vectors" --tsv > "$work/requests.tsv"
test -f "$pulp_repo/go.mod";git -C "$pulp_repo" cat-file -e "$pulp_commit^{commit}";mkdir "$work/pulp" "$work/pinned";git -C "$pulp_repo" archive "$pulp_commit" | tar -x -C "$work/pinned"
(cd "$work/pinned" && go build -buildvcs=false -o "$work/pulp-runner" ./cmd/pulp-seme-function-proof);cp "$repo/targets/wasm/pulp-function-v2/pulp.cell.toml" "$work/pulp/pulp.cell.toml";cp "$work/program.wasm" "$work/pulp/pure-function.wasm"; : > "$work/audit.tsv";tab=$(printf '\t')
while IFS="$tab" read -r kind name request response;do if [ "$kind" = valid ];then "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 -request "$request" > "$work/pulp-$name.log" 2>&1;rg -q "\"response\":\"$response\"" "$work/pulp-$name.log";value=$(node -e 'const b=Buffer.from(process.argv[1],"hex");console.log(b.readBigInt64LE().toString())' "$response");printf 'valid\t%s\t%s\n' "$name" "$value" >> "$work/audit.tsv";else if "$work/pulp-runner" -manifest "$work/pulp/pulp.cell.toml" -provider seme.function-v2 -request "$request" > "$work/pulp-$name.log" 2>&1;then exit 1;fi;printf 'malformed\t%s\trejected\n' "$name" >> "$work/audit.tsv";fi;done < "$work/requests.tsv"
node "$repo/reference/js/javascript-uab-04-pulp-audit.mjs" "$vectors" "$work/audit.tsv"
echo 'JavaScript UPB-04 typed cross-module parity: native, canonical, Wasm, and pinned Pulp agree'
