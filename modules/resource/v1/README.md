# Resource Contract v1

Resource v1 (`...6000@...6001`) describes a metadata-only resource manifest.
Resources have logical identity, Package-v4 ownership, a detached source-unit
reference, normalized path, text-or-bytes representation, canonical media type,
byte size, and SHA-256 digest. Placements map resources to destinations.

The contract contains no raw bytes, filesystem operations, decoders, storage
modes, or runtime loading semantics. Instance validation binds source-unit
metadata to an authenticated Project source inventory and a detached,
digest-addressed byte bundle.
