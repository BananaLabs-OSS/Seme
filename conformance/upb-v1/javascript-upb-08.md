# JavaScript UPB-08 evidence

Status: claimed 2026-09-11 by `scripts/check-javascript-upb-08.sh`.

The cumulative native ES-module project now includes project-owned transport
command/event records plus pure dispatch and replay functions. The neutral
selection resolves their exact semantic identities and package ownership from
Project v10. Ordered Transport v1 binds one stream, command kind, event kind,
port, ordered receive/send operations, capabilities, canonical framing,
sequence/correlation policy, replay limits, and exact frame bounds; Project v11
binds that plan to the complete prior project authority.

Native JavaScript, canonical Seme, standalone Wasm, and pinned Pulp agree on
dispatch behavior for normal, zero, and signed-i64 wrapping cases. Malformed
record boundaries reject. Neutral host runtime tests prove exact framing and
transport placement mechanics, while pinned Pulp remains explicitly a pure
synchronous carrier rather than a falsely claimed authoritative stream port.

Identical inputs reproduce Ordered-Transport-v1 and Project-v11. Missing
owners/functions, identical dispatch/replay selection, duplicate kinds,
mechanism-specific unknown fields, stale Project-v10 authority, and output
collisions reject atomically. The complete UPB01–UPB07 chain remains green with
the fifth native JavaScript module included.
