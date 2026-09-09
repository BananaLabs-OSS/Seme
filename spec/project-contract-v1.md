# Project Contract v1

Project Contract v1 describes an immutable collection of canonical packages
and its executable view without embedding source-language or build-system
mechanics. Its collision-audited module identity is `...e000`, revision
`...e001`.

`ProjectIdentity` contains the stable project name. `ProjectSnapshot` contains
that identity, a content-derived revision digest as bytes, an ordered nonempty
list of Package Contract v1 `Package` references, one root Package contained in
that list, and one Core Execution v35 `ExecutableProgram` reference.

The module uses Kernel `Import` entities to pin Package Contract v1
`...b000@...b001` and Core Execution v35 `...9000@...9023`. The snapshot's
content revision is produced from canonical snapshot content; it is never an
editor sequence number, timestamp, or source-control label.

This version defines structure only. Providers do not yet emit snapshots, and
it does not define package discovery, dependency resolution, workspace layout,
or projection policy.

## Unclaimed snapshot invariants

No ProjectSnapshot conformance claim is made yet. A subsequent instance
validator must reject an empty package list, a root outside that list, a
Program whose functions are not owned by the listed package contracts,
duplicate Package identities, unordered package references, and a revision
that is not the specified digest of immutable canonical snapshot content.
Until that validator and its malformed fixtures land, v1 proves the module's
schema, imports, identities, generation, and wire validity only.
