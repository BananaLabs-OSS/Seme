# Core Execution v60: typed native Go built-ins

Core Execution v60 retains ordinary Go `make`, `append`, `delete`, `copy`,
`new`, `cap`, and `clear` operations as explicit typed native invocations inside
otherwise canonical Seme programs.

Arguments are evaluated in source order and materialized at the exact Go types
required by each built-in. Result types, allocation behavior, collection
mutation, ellipsis expansion, and Go runtime ownership remain visible native
mechanics rather than being imported into neutral Core.

This milestone does not treat `defer` as an immediate call. Deferred execution
requires its own control semantic and remains an explicit native island until
that schema exists. No schema is added beyond v59.
