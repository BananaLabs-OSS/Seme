# Go UPB-05 acceptance design

Status: fixture foundation only; UPB-05 remains unclaimed.

The cumulative ordinary Go fixture is materialized by
`scripts/materialize-go-upb05-fixture.sh`. It retains the byte-pinned UPB-04
three-package source and 2,048-command native corpus, then adds typed
`configuration` and `service` packages plus a `policy` initializer.

Configuration is pure explicit input. Defaults and validation produce typed
results. Initialization order is observable as ordinary lifecycle state:
configuration produces stage 1, policy requires stage 1 and produces stage 2,
and service requires stage 2 before producing stage 3/ready. There are no
environment or filesystem reads, package `init` functions, mutable globals,
clock reads, or random sources.

The eventual authoritative gate `scripts/check-go-upb-05.sh` must establish:

1. The untouched materialized project passes the original 2,048 cases and the
   bounded defaults, validation, ordering, readiness, and immutability matrix.
2. Configuration v1 and Project v6 instances authenticate exact canonical
   types, defaults, validators, initializer functions, lifecycle transitions,
   ownership, dependencies, and source origins above Project v5.
3. Native and canonical observations agree for the cumulative command and
   configuration corpus.
4. A source-free deterministic Project v6 report exposes every binding,
   default, constraint, initialization edge, lifecycle state, and realization
   boundary without manufacturing ambient configuration semantics.
5. Standalone Wasm and pinned Pulp agree with the same canonical graph and ABI
   observations.
6. Authenticated projection rebuilds as an ordinary Go project, retains opaque
   module/checksum/test bytes, and re-lifts to exact canonical meaning.
7. Wrong-typed or invalid defaults, missing/duplicate/conflicting bindings,
   initializer cycles, missing predecessors, private or wrong-signature
   functions, order/readiness violations, ambient reads, stale origins,
   dependency or graph tampering, malformed ABI, denied capabilities,
   symlinks, and existing destinations reject without partial output.

Expected cumulative bundle additions are `configuration-v1.seme` and
`project-v6.seme`; `COMPLETE.sha256` must cover those and all nine UPB-04
artifacts. Scorecard and claimed-runner changes are forbidden until the full
seven-class gate passes.
