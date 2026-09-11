# JavaScript UPB-05 acceptance design

Status: accepted implementation design; unclaimed.

JavaScript UPB-05 extends the cumulative typed Project-v4 fixture with the
existing neutral Configuration-v3 and Project-v8 authorities. JavaScript owns
the native module syntax and JSDoc presentation; configuration values,
validation, dependency ordering, lifecycle transitions, and revisions remain
language-neutral.

The bounded fixture declares typed `Enabled: bool`, `Limit: i64`, and
`Namespace: string` values with explicit defaults and one selected override.
Three separately owned initializer functions form an acyclic declared order.
Each initializer receives its configuration explicitly and returns its next
lifecycle state; none may read `process.env`, global variables, filesystem
configuration, current time, or other ambient host state.

The adapter must reconcile the configuration manifest against canonical
function/type identities and Package-v2 ownership, then execute the validated
plan through the neutral configuration executor. Original and projected native
modules, canonical evaluation, standalone Wasm, and pinned Pulp must agree on
the cumulative observation corpus. Reordered declarations with equivalent
meaning reproduce authority; changed selected values change the configuration
and Project revisions.

Missing/duplicate values, wrong types, invalid defaults, unknown selections,
missing initializer dependencies, cycles, package-owner disagreement, ambient
reads, stale/tampered authority, and output collisions reject before any
initializer or publication becomes observable.

Only `scripts/check-javascript-upb-05.sh` may claim the cell. It must run the
complete UPB-04 gate and prove all seven evidence classes for the resulting
Project-v8 chain.
