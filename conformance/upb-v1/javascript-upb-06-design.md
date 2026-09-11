# JavaScript UPB-06 acceptance design

Status: accepted implementation design; unclaimed.

JavaScript UPB-06 extends the exact cumulative Project-v8 lifecycle with an
explicit text resource and opaque binary resource. Resource discovery and byte
acquisition remain outside JavaScript semantics: the provider supplies a
deterministic source snapshot, while neutral resource machinery validates
paths, sizes, digests, media types, package ownership, placements, and detached
content before emitting Resource v1 and Project v9.

Projection must reproduce native JavaScript modules, opaque project metadata,
the resource selection, and both resource byte streams exactly. Original and
projected native behavior and canonical meaning must agree. Equivalent inputs
must reproduce Resource-v1 and Project-v9 authorities.

Traversal, symlinks, missing or changed bytes, invalid UTF-8 text, digest or
size mismatch, duplicate identities/destinations, over-limit input, stale or
mixed Project-v8 authority, detached-blob tampering/orphans, malformed
manifests, and output collisions must reject without publication.

Only `scripts/check-javascript-upb-06.sh` may claim the cell. It must run the
complete UPB-05 gate and prove all seven evidence classes.
