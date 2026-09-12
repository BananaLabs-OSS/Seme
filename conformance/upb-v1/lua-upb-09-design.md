# Lua UPB-09 acceptance design

Status: claimed 2026-09-12 by `scripts/check-lua-upb-09.sh`.

Lua UPB-09 extends Project v11 with provider-neutral controlled effects.
Ordinary Lua owns record/function declarations for explicit clock samples,
seeded random transitions, application dispatch, and effect-free replay. A
`Seme.observe(true)` statement requests one Boolean observability effect; the
project graph assigns its canonical identity to the declaring package.

Core does not acquire clocks, randomness, logging backends, or ambient Lua
runtime behavior. Controlled Effects v1 describes injected inputs, deterministic
transition policy, capability requests, replay behavior, bounds, and placement.
Concrete delivery remains an authorized host mechanic.

Acceptance requires deterministic Project-v12 authority, native/canonical/Wasm/
Pulp value-and-effect parity, explicit host and placement conformance, complete
cumulative projection/re-lift, and atomic rejection of invalid ownership,
callables, policies, bounds, ambient-provider fields, stale authority, and
collisions. Only `scripts/check-lua-upb-09.sh` may claim this cell.
