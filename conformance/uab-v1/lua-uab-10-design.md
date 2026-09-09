# Lua UAB-10 semantic design and evidence

Lua source may carry a sealed `---@seme-id 80…` declaration identity. The
identity is metadata for Seme re-import, not a Lua runtime behavior. Formatting
does not affect it, and projection emits it so an identity-addressed canonical
rename can change the native spelling without changing the semantic entity.

The bounded adapter accepts only lowercase 128-bit provider identities in the
provider-owned `80` namespace. Invalid and duplicate identities are rejected
with source locations. The UAB-10 gate proves identical lift across formatting,
identity-preserving canonical rename, executable projection and byte-identical
re-import, exact native/canonical/Wasm/Pulp Boolean observations, malformed wire
rejection, and rejection of forged or duplicate identity metadata.
