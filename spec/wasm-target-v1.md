# Wasm target v1

Wasm target v1 is the first artifact-producing backend for a checked Seme Target
Contract plan. Its deliberately finite profile requires:

- one executable `wasm32-pulp-v1` plan;
- two selected `adapted` resolutions for `go:log` and `observability.log`;
- one `seme.pulp.log-v1` Boundary;
- one canonical Core Function with three modular signed-i64 parameters;
- body semantics exactly `parameter[0] + parameter[1] <= parameter[2]`;
- BooleanType result.

The backend independently validates those canonical entities and emits a
deterministic 205-byte WebAssembly Pulp reactor:

```text
import pulp.log_bool(i32) -> i32
export admit(i64, i64, i64) -> i32
export memory
export pulp_alloc / pulp_init / pulp_step / pulp_shutdown
```

The imported function is the target realization of the explicitly adapted
logging effect. It observes the same decision returned by `admit`. The Wasm
contains no Go runtime, source, AST, or package machinery.

The host import returns a status code; nonzero status traps so denial cannot
silently discard the observable effect.

The current backend implementation is a small Go-hosted target component. Go is
used to implement the emitter, not as semantic authority: lowering reads only
the canonical Seme graph and rejects unsupported graph shapes. This is an
implementation layer that can later be lifted/self-hosted without changing the
target contract or artifact behavior.

Conformance executes the artifact in Node's WebAssembly engine with a host
adapter implementing the declared import. Five result/effect traces, including
signed overflow, match the ordinary Go package. A valid exact-only plan rejects
before artifact production. The Node adapter proves the ABI and effect mapping;
the same reactor is also executed by actual Pulp in the subsequent target proof.
