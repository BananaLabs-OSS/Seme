# Semantic Module Registry v1

The Semantic Module Registry audits ownership and lineage across authoritative
`modules/<family>/<revision>/module.g1` artifacts. Its gate verifies the
tracked G1 and canonical-wire digests and then parses the narrow ownership
envelope from G1. Reproducing and semantically validating `module.seme` from
G1 remains the responsibility of each module's generation gate. A module may
reference declarations supplied by exact imports, so isolated Kernel
validation without its resolution closure is not a registry claim. The
registry parser is not a second canonical schema validator.

Globally owned identities are:

- module identities;
- revision identities; and
- identities named by the Module declaration's exports list.

The same module and exported identities may recur across revisions in one
module family. They may not be owned by another family. Non-exported entity
identities are revision-local implementation details and may coincide across
families; the registry deliberately does not assign them global ownership.
Ownership is scoped by role because canonical identities are contextual: the
same bytes may validly occur in a module, revision, and entity position.
Collisions reject within the module role, revision role, or exported-entity
role across lineages; byte reuse across different roles is permitted.

Every artifact must have one module declaration and a nonempty, duplicate-free
exports list whose members exist in that artifact. Module Import entities must
be referenced exactly once by the declaration and pin an existing module and
revision pair. Revision parent pins must exist in the same module lineage, be
unique and sorted, and form an acyclic graph. Duplicate artifacts for one
module/revision pair and duplicate revision artifacts reject.

The deterministic report is
`conformance/semantic-module-registry-v1.json`. It records families, revisions,
parents, exports, imports, entity counts, and artifact digests. The gate is
`scripts/check-semantic-module-registry.sh`.

The registry does not currently compare exported entity schema or mechanic
definitions across revisions. Such drift requires a separately specified
compatibility policy; successful ownership audit must not be described as
schema-version compatibility evidence.
