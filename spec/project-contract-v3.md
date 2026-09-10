# Project Contract v3

Project Contract v3 is revision `...e003` of module `...e000`, an additive
child whose sole parent is Project Contract v2 revision `...e002`. Its Module
declaration is version 3. It preserves every v1 and v2 exported identity,
pins Package Contract v2 `...b000@...b002`, and continues to pin Core Execution
v35 `...9000@...9023`.

V3 adds two neutral schemas:

- `PackageGraphBinding` (`...e017`) binds one Package Contract v2
  `PackageGraph` to one Project Contract v2 `SourceInventory` and carries an
  independent content-revision byte string.
- `ProjectGraphSnapshot` (`...e018`) binds one existing `ProjectSnapshot` to
  one `PackageGraphBinding` and carries its own content-revision byte string.

Their fields are:

| Field | Identity | Kind |
|---|---|---|
| package graph | `...e170` | reference to `...b020` |
| source inventory | `...e171` | reference to `...e016` |
| binding content revision | `...e172` | bytes |
| project snapshot | `...e180` | reference to `...e011` |
| package-graph binding | `...e181` | reference to `...e017` |
| project-graph content revision | `...e182` | bytes |

The direction of these references deliberately avoids a Package/Project
schema cycle. Package v2 `SourceOrigin.source_unit_identity` remains
unconstrained inside Package Contract; a future v3 instance validator will
resolve those identities against the bound SourceInventory and validate path,
digest, membership, graph ownership, and both content revisions.

This milestone defines vocabulary only. No provider, Project emitter, package
detail adapter, or build command currently emits a Project v3 instance, and
the generated module is not evidence of that future integration.
