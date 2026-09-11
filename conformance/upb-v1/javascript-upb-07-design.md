# JavaScript UPB-07 acceptance design

Status: claimed 2026-09-11 by `scripts/check-javascript-upb-07.sh`.

JavaScript UPB-07 extends the cumulative Project-v9 authority with one neutral,
capability-backed durable-state family. Ordinary native JavaScript declares two
JSDoc-authenticated record schemas, pure validators, and exactly one directed
v1-to-v2 migration. The existing language-neutral durable adapter resolves
those declarations by package ownership and semantic type/function identity;
neither Core nor Durable State v1 recognizes JavaScript syntax, field names,
or fixture identities.

The bounded family uses canonical Seme encoding, a string key, signed-i64
domain errors represented natively as JavaScript `bigint`, explicit Load and
CompareAndSwap capabilities, opaque comparison tokens, and fixed payload/key
bounds. Filesystems and databases remain unclaimed provider realizations. The
stateful DurablePort conformance adapter remains an explicit host boundary;
pinned Pulp executes only the pure migration graph because it has no certified
opaque-token CAS provider.

Project v10 must bind the exact Project-v9 graph, Durable-State-v1 plan, and a
valid Source-Presentation-v1 manifest. This JavaScript cell has no additional
source aliases, so the presentation manifest is intentionally empty rather
than inventing Go alias semantics. Native ES-module projection and exact
resource preservation remain covered cumulatively by UPB-06.

Acceptance requires:

1. Identical authenticated inputs reproduce Durable-State-v1,
   Source-Presentation-v1, Project-v10, and the completion manifest.
2. Native JavaScript, canonical Seme evaluation, standalone Wasm, and pinned
   Pulp agree on the v1-to-v2 migration for zero, positive, and negative i64
   fields; malformed record boundaries reject canonically.
3. The neutral DurablePort host conformance suite proves ordered Load/CAS,
   authorization, bounds, opaque-token threading, conflicts, and atomic writes
   without ambient filesystem/database behavior.
4. Missing owners/callables, a validator substituted for the migration, wrong
   codec, unknown selection fields, stale Project-v9 or Resource-v1 artifacts,
   and output collisions reject without publishing a partial Project-v10.
5. The complete JavaScript UPB-06 gate remains green, including native source
   projection/re-lift and native/canonical/Wasm/Pulp application behavior.

Only `scripts/check-javascript-upb-07.sh` may claim this cell.
