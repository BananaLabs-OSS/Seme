# Lua UPB-06 acceptance design

Status: implemented and claimed by `scripts/check-lua-upb-06.sh` on
2026-09-12.

The cumulative Lua project declares one UTF-8 text resource and one binary
resource with stable identities, source paths, destinations, media types,
sizes, and SHA-256 digests. Neutral Resource v1 records their meaning and
Project v9 binds that authority to the exact configured Project-v8 snapshot.
Filesystem traversal and copying remain target mechanics.

Composition emits digest-addressed detached blobs. Repeating it must reproduce
the complete artifact directory. Modular Lua projection copies all resource and
selection bytes exactly, then rebuilds Configuration v3, Project v8, Resource
v1, Project v9, blobs, and the semantic resource report independently.

The sole gate inherits UPB-01 through UPB-05 and rejects changed bytes,
traversal, digest or size mismatch, duplicate destinations, unknown fields,
symlink substitution, stale Project-v8 authority, and publication collisions
without partial output.

This cell does not claim arbitrary asset pipelines, decoding, streaming, or
runtime filesystem authority.
