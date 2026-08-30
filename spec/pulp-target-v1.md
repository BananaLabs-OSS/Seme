# Pulp target v1

Pulp target v1 is the first execution of a Seme-derived artifact by the actual
Pulp runtime.

The scoped Wasm backend emits a Pulp reactor with:

- linear memory;
- `pulp_alloc`, `pulp_init`, `pulp_step`, and `pulp_shutdown` exports;
- `pulp_on_call` implementing provider `quota.admit-v1`;
- `pulp.log_bool(i32) -> i32` as its only host import.

Each provider request carries fixed numeric fields plus bounded subject and
evidence data through Application Wire v2. Responses carry either `ResultOk`
with the Boolean decision and preserved fields or the source-derived
`ResultError` message. A cell is loaded once and handles repeated calls. The imported effect
returns zero when granted; any nonzero status traps, ensuring a capability
denial cannot silently erase an observable effect.

The Pulp repository supplies a small deployment binary,
`cmd/pulp-seme-proof`, which registers provider `seme.pulp.log-v1` for
capability `observability.log` and otherwise uses the normal Pulp runtime.
That deployment boundary is committed in Pulp through `c303c74`.
The allowed manifest proves the real manifest loader, capability registry,
wazero instantiation, allocation/configuration, init, step loop, effect call,
signal handling, shutdown, and clean exit. The denied manifest omits the
capability; Pulp binds its gated stub, the guest traps on status `99`, and cell
initialization fails without performing the effect.

This proves actual Pulp request/response execution and capability enforcement
for one schema-derived record, both Result variants, and one scalar effect. It
is not yet a general Pulp target adapter, recursive codec, Component Model package,
network input route, or production logging extension.
