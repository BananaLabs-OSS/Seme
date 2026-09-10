# Go UPB-03 evidence draft

The candidate acceptance gate is `scripts/check-go-upb-03.sh`. This document
does not claim UPB-03 complete; the scorecard and evidence-gate registry remain
unchanged until the gate and its applicability adversaries are independently
reviewed.

The gate exercises seven evidence classes over the bounded offline-dependency
fixture:

1. Native Go vectors execute using only the checked file proxy.
2. Dependency v1 resolution is exact, pinned, repeatable, and inspectable
   without source bytes.
3. Project v4 authenticates the Dependency closure and Project v3 snapshot.
4. Canonical observations match the native vectors.
5. Standalone Wasm and the pinned Pulp runtime match those observations.
6. Strict native projection preserves `go.mod`, `go.sum`, canonical behavior,
   dependency coordinates, and repeatable re-lift identity. Local source
   attestations visibly change when projection rewrites source presentation.
7. Floating, replaced, undeclared, checksum-invalid, archive-tampered,
   symlinked, graph-tampered, unrelated-but-valid, and existing-output cases
   fail atomically without partial evidence or bundles.

The Go-specific applicability check binds the otherwise language-neutral
Dependency closure to exact Project v3 package/import/source identities and
the byte-preserved `go.mod`. This check is separate from the Dependency and
Project contracts and must reject a valid closure belonging to another
project.
