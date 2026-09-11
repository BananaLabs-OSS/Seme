# JavaScript UPB-02 acceptance design

Status: claimed 2026-09-11 by `scripts/check-javascript-upb-02.sh`.

JavaScript UPB-02 extends the claimed Project-v2 source authority with a real
ECMAScript module graph under the existing neutral Package-v2 and Project-v3
contracts. It must not flatten modules into one synthetic source file when
describing ownership, even when the execution backend later fuses their code.

## Native module model

Each normalized relative `.js` path is one source unit and ES module. Named
relative imports resolve using the bounded ECMAScript path algorithm already
used by the JavaScript package provider. Declarations retain module ownership,
source spans, exported/private visibility, and stable semantic identities.
Import evidence records source module, local alias, imported name, target
module, target declaration, and exact source location.

V1 supports static named relative imports only. Default imports, namespace
imports, dynamic imports, bare package specifiers, re-exports, CommonJS, and
package export maps remain unsupported rather than receiving approximate
meaning. UPB-03 owns bounded external package resolution.

## Required validation

The adapter must reject duplicate normalized paths, missing modules, missing or
private imported declarations, duplicate bindings, declaration collisions,
imports that escape the project, self-imports, and module cycles with native
locations. Input enumeration order cannot affect canonical output.

Package-v2 detail is derived from independently parsed module ASTs and the
provider's canonical declaration mappings. Project-v3 must bind the exact
Project-v1 semantics, Project-v2 source inventory, and Package-v2 graph.
Canonical execution may be physically fused, but ownership and dependencies
remain separate and authenticated.

## Claim rule

Only `scripts/check-javascript-upb-02.sh` may claim the cell. It must invoke the
complete JavaScript UPB-01 gate, prove deterministic graph construction and
reporting, execute native/canonical/Wasm/Pulp behavior, project a functioning
multi-module native project, re-lift deterministically, and exercise missing,
private, cyclic, malformed, tampered, and output-collision adversaries across
all seven evidence classes.
