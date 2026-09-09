#!/bin/sh
set -eu

repo=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/seme-package-v2.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

module="$repo/modules/package/v2"
k0="$repo/bootstrap/seme-k0-linux-amd64"
(
  cd "$repo/reference/go"
  GOCACHE="$work/go-cache" go run ./cmd/package-module --version 1 > "$work/v1.g1"
  GOCACHE="$work/go-cache" go run ./cmd/package-module --version 2 > "$work/v2.g1"
)
cmp "$repo/modules/package/v1/module.g1" "$work/v1.g1"
cmp "$module/module.g1" "$work/v2.g1"
"$k0" "$repo/compiler/g1-compiler.k0" "$work/v2.g1" "$work/v2.seme"
cmp "$module/module.seme" "$work/v2.seme"
(cd "$module" && sha256sum -c module.g1.sha256 && sha256sum -c module.seme.sha256)
"$k0" "$repo/compiler/kernel-wire-validator.k0" "$module/module.seme"
"$k0" "$repo/modules/foundation/v1/validator.k0" "$module/module.seme"

# The authoritative registry verifies the shared module lineage, v1 parent,
# export ownership, and deterministic report including this revision.
"$repo/scripts/check-semantic-module-registry.sh"
node -e '
const report = require(process.argv[1]);
const family = report.families.find((item) => item.family === "package");
if (!family || family.module !== "0000000000000000000000000000b000") process.exit(1);
const v1 = family.revisions.find((item) => item.revision === "0000000000000000000000000000b001");
const v2 = family.revisions.find((item) => item.revision === "0000000000000000000000000000b002");
if (!v1 || !v2 || v2.parents.length !== 1 || v2.parents[0] !== v1.revision) process.exit(1);
for (const identity of v1.exports) if (!v2.exports.includes(identity)) process.exit(1);
for (const identity of ["b020","b021","b022","b023","b024","b025","b026","b200","b268"].map((x) => x.padStart(32,"0"))) if (!v2.exports.includes(identity)) process.exit(1);
' "$repo/conformance/semantic-module-registry-v1.json"

echo 'Package Contract v2: additive lineage, deterministic generation, canonical compilation, and registry validation pass'
