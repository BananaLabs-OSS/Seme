# Lua UPB-03 acceptance design

Status: implemented and claimed by `scripts/check-lua-upb-03.sh` on
2026-09-12.

The Lua provider reads a deliberately static rockspec and a strict versioned
lock. It resolves one project-local module and one external LuaRocks-style
package tree from explicit offline roots. Exact versions, sources, tree bytes,
and integrity are required before it emits the language-neutral Dependency-v1
closure. LuaRocks syntax and resolution policy do not enter Core.

Project v4 binds this closure to the preceding source inventory and module
graph. Projection replaces semantic Lua sources while preserving the rockspec,
lock, and package manifest byte-for-byte; an offline repeated resolution of the
projected tree must reproduce the same closure.

The sole gate inherits UPB-01 and UPB-02, then checks resolver unit adversaries,
deterministic Dependency-v1 and Project-v4 artifacts, native/canonical/Wasm/
Pulp parity, and atomic rejection of floating requirements, substitutions,
unknown fields, unsafe paths, integrity drift, tampered authority, and output
collisions.

This is a bounded static LuaRocks-style proof. It does not execute arbitrary
rockspec code, contact a registry, or claim general LuaRocks compatibility.
