# Lua UPB-09 evidence

Status: claimed 2026-09-12 by `scripts/check-lua-upb-09.sh`.

The cumulative Lua project adds native controlled-effect records, clock and
seeded-random functions, application dispatch/replay, and an owned Boolean
observation request. Controlled Effects v1, its deterministic replay artifact,
and Project v12 reproduce from authenticated Project-v11 inputs.

Native Neovim Lua captures exactly one requested observation per accepted
dispatch. Canonical Seme, standalone Wasm, and pinned Pulp produce the same
values and ordered effect traces, including signed-i64 wraparound. Independent
runtime and placement suites verify authorization without claiming a clock,
random source, logger, or network provider inside Seme Core.

Every earlier Lua project gate remains green. Missing owners/functions, replay
aliasing, unapproved algorithms, invalid bounds, ambient-provider fields, stale
Project-v11 authority, and destination collisions reject without partial
publication.
