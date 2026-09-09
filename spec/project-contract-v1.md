# Project Contract v1

Project Contract v1 describes an immutable collection of canonical packages
and its executable view without embedding source-language or build-system
mechanics. Its collision-audited module identity is `...e000`, revision
`...e001`.

`ProjectIdentity` contains the stable project name. `ProjectSnapshot` contains
that identity, a 32-byte SHA-256 content revision, an ordered nonempty
list of Package Contract v1 `Package` references, one root Package contained in
that list, and one Core Execution v35 `ExecutableProgram` reference.

The module uses Kernel `Import` entities to pin Package Contract v1
`...b000@...b001` and Core Execution v35 `...9000@...9023`. The snapshot's
content revision is produced from canonical snapshot content; it is never an
editor sequence number, timestamp, or source-control label.

This version defines structure only. The reference `projectemitter` can now
emit a canonical instance from a validated Execution graph and explicit typed
package metadata. Language providers do not yet produce that metadata
automatically, and the contract does not define package discovery, dependency
resolution, workspace layout, or projection policy.

The emitter resolves and pins the immutable Execution v35, Package v1, and
Project v1 contracts before construction. Package versions commit to canonical
interface, dependency, and reachable function semantics rather than source
filenames or comments. Project snapshot revisions and enclosing artifact
revisions are separate content-derived values; neither uses an editor sequence
number. Generated Project-instance identities use an explicitly role-scoped
domain and do not replace provider-owned declaration identities.

## Snapshot digest and validation

The revision is SHA-256 over the domain bytes `project-snapshot-v1` followed by
a zero byte, the raw 16-byte identity, each ordered raw Package identity, the
root and Program identities, and deterministic typed encodings for the
identity, every listed Package, the Program, and all entities transitively
referenced by them. Thus changing package metadata, Program membership, or
function semantics invalidates a stale revision. The instance validator
requires exactly one snapshot and requires the envelope's own Module entity to
contain a Kernel Import resolving Project Contract `...e000` exactly to
revision `...e001`; ancestry alone is not authority. It also requires sorted
unique and nonempty Package references, the root
exactly once in that list, Package v1 and ExecutableProgram schemas on
referenced entities, and the matching digest.
Package-to-function ownership remains a Package Contract concern and is not
claimed by this structural validator.

The validator consumes canonical `.seme` wire bytes through the shared typed
wire decoder, not construction G1 text. Value tags, entity versions,
record members, list items, references, and hole identities participate in the
digest. Duplicate entity fields or record members and malformed envelope or
count framing reject during wire decoding before project semantics run.
