# Wasm target v1

Wasm target v1 is the first artifact-producing backend for a checked Seme Target
Contract plan. Its deliberately finite profile requires:

- one executable `wasm32-pulp-v1` plan;
- two selected `adapted` resolutions for `go:log` and `observability.log`;
- one `seme.pulp.log-v1` Boundary;
- one canonical Core Function accepting an `AdmitRequest` RecordType;
- ordered `Current`, `Delta`, and `Limit` signed modular-i64 fields;
- body semantics exactly `request.Current + request.Delta <= request.Limit`;
- an `AdmitResponse` RecordType containing an `Accepted` Boolean field.

The backend independently validates those canonical entities and emits a
deterministic 262-byte WebAssembly Pulp reactor:

```text
import pulp.log_bool(i32) -> i32
export memory
export pulp_alloc / pulp_init / pulp_step / pulp_shutdown
export pulp_on_call(name, request, response-out) -> status
```

The imported function is the target realization of the explicitly adapted
logging effect. It observes the same decision returned in the provider response. The Wasm
contains no Go runtime, source, AST, or package machinery.

The host import returns a status code; nonzero status traps so denial cannot
silently discard the observable effect.

The current backend implementation is a small Go-hosted target component. Go is
used to implement the emitter, not as semantic authority: lowering reads only
the canonical Seme graph and rejects unsupported graph shapes. This is an
implementation layer that can later be lifted/self-hosted without changing the
target contract or artifact behavior.

Application Wire v1 derives field offsets and total sizes from those canonical
record entities. Its v1 scalar mapping encodes signed i64 fields little-endian
and Boolean fields as one byte. The backend constructs Wasm loads, stores, and
length checks from the derived layout; Pulp carries the resulting bytes without
interpreting their semantics.

Conformance executes the artifact in Node's WebAssembly engine with a host
adapter implementing the declared import. Five request/result/effect traces, including
signed overflow, match the ordinary Go package. A valid exact-only plan rejects
before artifact production. The Node adapter proves the ABI and effect mapping;
the same reactor is also executed by actual Pulp in the subsequent target proof.
