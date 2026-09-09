# JavaScript UAB-08 semantic design

This bounded cell uses explicit `Seme.Result<bigint,bigint>` values and a total
`Seme.matchResult` branch as the JavaScript spelling of declared fallibility and
error propagation. Error code `99n` is explicit, and the error arm returns it unchanged; it is a
canonical propagation decision, not a JavaScript exception or truthiness test.

One source declares `CheckPositive` and `IncrementPositive`. Shared vectors
distinguish success, zero and negative errors, unchanged error payloads, and
modular overflow after a successful check.

JavaScript exceptions, rejected Promises, `undefined`, `null`, falsy sentinels,
and coercive error conventions are not treated as canonical Results. They need
explicit adapters or native-island semantics.
