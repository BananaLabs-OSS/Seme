# Lua UPB-07 acceptance design

Status: claimed 2026-09-12 by `scripts/check-lua-upb-07.sh`.

Lua UPB-07 extends the cumulative Project-v9 authority with one neutral,
capability-backed durable-state family. Ordinary native Lua declares two
annotated record schemas, pure validators, and exactly one directed v1-to-v2
migration. The project-neutral durable adapter resolves those declarations by
package ownership and semantic identity; Core and Durable State v1 contain no
Lua syntax, fixture names, or provider behavior.

The bounded family uses canonical Seme encoding, string keys, signed-i64 domain
errors, explicit Load and CompareAndSwap capabilities, opaque comparison
tokens, and fixed payload/key bounds. Filesystems and databases remain
unclaimed. The stateful DurablePort conformance adapter is an explicit host
boundary; pinned Pulp executes the pure migration graph, not a storage service.

Project v10 must bind the exact Project-v9 graph, Durable-State-v1 plan, and a
valid Source-Presentation-v1 manifest. Native projection preserves all earlier
source, configuration, dependency, and resource authority.

Acceptance requires deterministic authority reproduction; native Lua,
canonical, Wasm, and Pulp migration parity; independent ordered Load/CAS host
tests; exact projection and re-lift; and atomic rejection of missing owners,
invalid callables, codecs, stale authorities, unknown fields, and collisions.
Only `scripts/check-lua-upb-07.sh` may claim this cell.
