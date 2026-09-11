# JavaScript UPB-06 evidence

Status: claimed 2026-09-11 by `scripts/check-javascript-upb-06.sh`.

The cumulative gate extends the exact JavaScript Project-v8 lifecycle with one
UTF-8 text resource and one opaque binary resource. A provider-owned source
snapshot and explicit selection feed language-neutral resource composition.
Resource v1 authenticates identities, package ownership, source units, paths,
media types, sizes, SHA-256 digests, placements, and detached blobs; Project v9
binds that manifest to the complete configured project.

Identical inputs reproduce. Projection preserves resource selections, package
metadata, and resource bytes exactly while reconstructed native ES modules
re-lift to identical canonical execution. The projected source snapshot may
legitimately change because formatting is provenance; its semantic resource
model and detached content remain identical.

The cumulative native/canonical/Wasm/Pulp behavior remains green. Changed or
missing bytes, traversal, digest and size drift, duplicate destinations,
unknown fields, symlinks, stale Project authority, malformed inputs, and output
collisions reject without publication.
