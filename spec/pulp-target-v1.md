# Pulp target v1

Pulp target v1 is the first execution of a Seme-derived artifact by the actual
Pulp runtime.

The scoped Wasm backend emits a Pulp reactor with:

- linear memory;
- `pulp_alloc`, `pulp_init`, `pulp_step`, and `pulp_shutdown` exports;
- the independently callable `admit(i64, i64, i64) -> i32` export;
- `pulp.log_bool(i32) -> i32` as its only host import.

`pulp_init` evaluates the canonical quota policy once with the conformance
vector `(40, 2, 50)`. The imported effect returns zero when granted. Any
nonzero status traps, ensuring a capability denial cannot silently erase an
observable effect.

The Pulp repository supplies a small deployment binary,
`cmd/pulp-seme-proof`, which registers provider `seme.pulp.log-v1` for
capability `observability.log` and otherwise uses the normal Pulp runtime.
That deployment boundary is committed in Pulp as `97b1c8c`.
The allowed manifest proves the real manifest loader, capability registry,
wazero instantiation, allocation/configuration, init, step loop, effect call,
signal handling, shutdown, and clean exit. The denied manifest omits the
capability; Pulp binds its gated stub, the guest traps on status `99`, and cell
initialization fails without performing the effect.

This proves actual Pulp execution and capability enforcement for one scalar
effect. It is not yet a general Pulp target adapter, request/response provider,
Component Model package, dynamic input route, or production logging extension.
