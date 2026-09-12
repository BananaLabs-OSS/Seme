# Lua UPB-08 evidence

Status: claimed 2026-09-12 by `scripts/check-lua-upb-08.sh`.

The cumulative native Lua project now contains distinct TransportCommand and
TransportEvent schemas plus Dispatch and Replay functions. The neutral
selection resolves those declarations by authenticated package ownership.
Ordered Transport v1 and Project v11 reproduce byte-for-byte from identical
inputs.

Native Neovim Lua, canonical Seme, standalone Wasm, and pinned Pulp agree on
zero, ordered, and signed-i64 wraparound dispatch vectors; malformed record
boundaries reject. Independent transport-host and placement suites prove the
runtime boundary without claiming a particular network protocol.

The complete UPB01–UPB07 source, modules, dependencies, typed calls,
configuration, resources, durable state, native projection, and exact re-lift
remain green. Missing owners/callables, invalid role aliasing, duplicate kinds,
unknown protocol selection, stale authority, and collisions reject atomically.
