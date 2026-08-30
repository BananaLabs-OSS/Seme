# Seme Application Proof v1

Application Proof v1 turns the fixed initialization demonstration into a
repeatable request/response application slice:

```text
ordinary Go quota policy
  -> Go provider and type checker
  -> canonical Seme function and effect plan
  -> independently generated Wasm reactor
  -> Pulp pulp_on_call provider
  -> structured request / Result response
```

The scoped request is a canonical Core Execution v3 `RecordType`. Application
Wire v1 derives this layout from its ordered `RecordField` entities:

| Offset | Field | Type |
|---:|---|---|
| 0 | `current` | little-endian signed i64 |
| 8 | `delta` | little-endian signed i64 |
| 16 | `limit` | little-endian signed i64 |

Application Wire v2 extends the header with subject/evidence lengths and appends
their bytes. The response begins with a Result tag, then the Boolean decision,
lengths, and preserved subject/evidence data. Each variable field is bounded to
4096 bytes. Malformed sizes return status 2. The provider name remains
`quota.admit-v1`, and the decision retains signed 64-bit modular addition. An
empty subject returns the lifted `AdmitError{"subject required"}` as
`ResultError` before the logging effect.

`./scripts/demo-application-v1.sh` loads one Pulp cell, sends four different
requests across both Result variants, prints their structured responses, and
then runs the identical Wasm
without `observability.log`. The denied call traps before the effect occurs.

The ordinary Go structs are lifted into canonical `RecordType`, `RecordField`,
`FieldRead`, and `RecordConstruct` entities. The backend independently checks
that graph and generates its Wasm request loads, response stores, and size
guards from the canonical field order and scalar widths.

The provider's source analysis is semantic within this bounded profile rather
than tied to the fixture's spelling. Conformance reverses the empty-string
comparison, renames the Boolean local, reorders keyed response fields, and
changes the error literal. Equivalent presentation choices still lift, the
changed literal is preserved, and the derived Wasm returns that changed text.

The decision now lives in an ordinary Go `WithinLimit` helper in a separate
source file. Core Execution v6 retains `Admit` and `WithinLimit` as separate
canonical Functions connected by `FunctionCall`; lowering emits two Wasm
functions and a real call instruction.

This milestone is usable as a bounded technical demo. Input is still driven by
the proof deployment instead of HTTP, Application Wire v2 supports only its
current scalar/string/bytes profile, and only the quota profile's current
statement and expression families are supported.
Those limitations are explicit rather than generalized into a claim of
arbitrary Go application support.
