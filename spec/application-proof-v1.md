# Seme Application Proof v1

Application Proof v1 turns the fixed initialization demonstration into a
repeatable request/response application slice:

```text
ordinary Go quota policy
  -> Go provider and type checker
  -> canonical Seme function and effect plan
  -> independently generated Wasm reactor
  -> Pulp pulp_on_call provider
  -> structured request / Boolean response
```

The scoped request is a canonical Core Execution v3 `RecordType`. Application
Wire v1 derives this layout from its ordered `RecordField` entities:

| Offset | Field | Type |
|---:|---|---|
| 0 | `current` | little-endian signed i64 |
| 8 | `delta` | little-endian signed i64 |
| 16 | `limit` | little-endian signed i64 |

The response is exactly one byte: zero for false and one for true. The provider
name is `quota.admit-v1`. Unsupported request sizes return status 2. The
underlying decision retains Go's signed 64-bit modular addition behavior.

`./scripts/demo-application-v1.sh` loads one Pulp cell, sends three different
requests, prints their structured responses, and then runs the identical Wasm
without `observability.log`. The denied call traps before the effect occurs.

The ordinary Go structs are lifted into canonical `RecordType`, `RecordField`,
`FieldRead`, and `RecordConstruct` entities. The backend independently checks
that graph and generates its Wasm request loads, response stores, and size
guards from the canonical field order and scalar widths.

This milestone is usable as a bounded technical demo. Input is still driven by
the proof deployment instead of HTTP, Application Wire v1 supports only signed
i64 and Boolean fields, and only the exact quota function shape is supported.
Those limitations are explicit rather than generalized into a claim of
arbitrary Go application support.
