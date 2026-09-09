# Project Source Discovery v1

Project Source Discovery v1 is a bounded host-side realization of the neutral
Project Contract v2 source inventory. It inventories every regular file below
one physical root and emits no executable semantics.

The caller supplies a stable project identity, explicit language, toolchain,
profile and profile semantic revision, plus a bounded classification policy.
Paths are normalized relative slash paths. Absolute paths, traversal,
backslashes, symlinks, non-regular files, ambiguous prefixes, digest drift and
configured count or size overruns reject.

Every file receives exactly one classification:

- tracked: a configured source extension outside ignored and vendored roots;
- ignored: a file under an explicitly ignored prefix or matching an explicitly
  ignored suffix (the Go UPB-01 profile uses this for `_test.go`);
- generated: a file beginning with the configured generated marker;
- vendored: a file under an explicitly vendored prefix; or
- opaque: every remaining regular file.

Tracked units have `semantic-projection` preservation. In this first bounded
profile all other units have `byte-exact` preservation. "Ignored" therefore
means ignored by semantic lifting, never omitted from the inventory or a round
trip.

The content revision is SHA-256 over a domain-separated, length-delimited
encoding of project identity, all four toolchain/profile fields, and the
path/classification/preservation/size/digest tuple for every path in bytewise
sorted order. It is distinct from the executable ProjectSnapshot semantic
revision. Source bytes remain outside canonical entities and are verified at
the source/projection boundary.

The reference implementation is `reference/go/projectsource`. Exact-copy
publication in `reference/go/projectroundtrip` validates the complete source
tree before staging, validates the staged tree before publication, refuses an
existing destination, and removes failed staging directories. This is a
bounded conformance primitive, not yet the semantic Go projector or a general
source-control implementation.

`reference/go/sourceinventory` emits the typed Project Contract v2 entities as
a separate canonical artifact. It embeds only the closed semantic
ProjectSnapshot reference closure needed to validate its binding; raw source
bytes are excluded. Its validator independently checks the exact Project v2
contract pin, semantic closure and revision, source revision, stable entity
identities, nonempty sorted units, closed class/preservation mappings,
toolchain membership, exports, and absence of orphan entities.

`reference/go/projectbundle` captures the bytes into a detached content store
with one sorted blob per distinct SHA-256 digest. Missing, changed, duplicate,
unsorted, and unreferenced blobs reject. The bundle remains valid after the
original tree is moved, proving that later projection need not trust mutable
original paths. The bundle is transport material and is not embedded in the
semantic Project or SourceInventory artifacts.
