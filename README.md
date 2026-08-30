# Seme

Seme is a canonical semantic programming platform. Syntax is a projection,
mechanics are composable semantic modules, packages and runtimes are explicit
dependencies, and execution targets declare how faithfully they realize those
semantics.

Seme is target-independent. WebAssembly is the first-class execution target and
primary portability proof, not the semantic foundation or the only backend.

This repository starts at the independent bootstrap boundary. USIR remains the
research prototype and behavioral reference; it is not copied into Seme and is
not part of Seme's trusted build path.

## Bootstrap direction

```text
frozen Linux-amd64 seed
    -> Hex0 source
    -> A0 canonical byte/integer notation
    -> S0 readable bootstrap notation
    -> Seme Kernel v1 reader/executor
    -> human-authored Seme compiler
    -> self-hosted Seme compiler
```

Only the first seed is native machine code with auditable assembly source.
Substantial software belongs in readable Seme, not assembly.

## Current status

- Structural Kernel v1 is frozen with pinned canonical artifacts, executable
  preservation/rejection/migration evidence, and deterministic diagnostics.
- The initial Hex0 seed is independently executable on Linux amd64.
- No Go, Rust, C, C++, Python, LLVM, or language runtime is required to run the
  frozen seed.
- The seed's checked-in hexadecimal image reproduces the frozen executable
  byte-for-byte without a host assembler or linker.
- The Hex0 seed constructs A0, and two A0 generations reproduce byte-identical
  A0 executables.
- A0 constructs the first portable K0-P0 executor; arithmetic, local-state,
  looping control flow, and malformed-image conformance fixtures pass.
- K0-P1 adds isolated function frames, parameters, calls, and entry selection
  while retaining P0 execution compatibility.
- K0-P2 adds bounded byte buffers and capability-scoped arguments, filesystem
  reads, and filesystem writes; a portable program performs a byte-identical
  file copy.
- P1/P2 images receive whole-image structural preflight before execution,
  including unreachable code and branch-witness validation.
- The first portable K0 compiler consumes checked A0 byte source and reproduces
  the K0 executor byte-for-byte; artifact construction has begun moving above
  native assembly.
- The portable S0 compiler consumes readable decimal/directive source and
  reproduces its own K0 image byte-for-byte across two generated generations.
- New bootstrap compiler work can now be authored in S0 without editing
  assembly, hexadecimal source, or A0 byte streams.
- The symbolic S1 compiler derives function tables, named calls, labels,
  instruction counts, code sizes, and byte witnesses. Its checked construction
  image and two self-produced generations are byte-identical.
- The first S1-authored Kernel wire validator accepts canonical nested semantic
  envelopes and rejects malformed integer, ordering, identity, tag, bounds, and
  trailing-byte cases.
- The S1-authored G1 compiler turns readable semantic graph notation into
  canonical Kernel wire. Its Counter and finite Kernel meta-schema outputs are
  byte-identical to the earlier A0 constructions and pass the independent
  validator.
- The candidate K0 semantic module represents programs, functions, and stable
  instruction identities in canonical Kernel graphs. The first S1-authored
  semantic lowerer turns its checked return-42 graph into a portable K0 image,
  derives execution layout rather than storing it canonically, and the frozen
  executor returns 42.
- The K0 semantic lowerer now has canonical Seme source and reproduces itself:
  checked compiler A, self-produced B, and B-produced C are byte-identical. Its
  S1 source is retained only as bootstrap history.
- Frozen Core 1 closes the Linux-amd64 development loop: the canonical lowerer,
  G1 frontend, Kernel validator, and migrator rebuild without S0, S1, assembly,
  Python, or another language toolchain. See
  [`compiler/FROZEN-CORE-1.md`](compiler/FROZEN-CORE-1.md).
- Host assembly tools remain optional construction and cross-check conveniences.
- The frozen compiler core is self-hosted. Semantic-foundation schemas continue
  above the frozen structural Kernel without expanding its wire boundary.
- The canonical located validator emits deterministic canonical diagnostic
  reports for every frozen rejection path and preserves valid inputs exactly.

See [`spec/architecture.md`](spec/architecture.md) and
[`spec/roadmap.md`](spec/roadmap.md) for the product proof sequence, and
[`bootstrap/README.md`](bootstrap/README.md) for the independent bootstrap.
The exact Kernel boundary and frozen hashes are recorded in
[`compiler/KERNEL-V1-FREEZE.md`](compiler/KERNEL-V1-FREEZE.md).
The broader Seme–registry–Pulp interoperability direction is documented in
[`spec/software-commons.md`](spec/software-commons.md).

The first ordinary-project dogfood layer is the narrow
[`Provider Contract v1`](spec/provider-contract-v1.md) Go proof. It imports
package functions, performs an identity-addressed semantic rename, minimally
projects resolved occurrences, runs the native Go tests, re-imports, and proves
identity recovery. It validates the provider boundary, not general Go support.
Run it with `./scripts/check-provider-v1.sh`.

The next proof, [`Core Execution v1`](spec/core-execution-v1.md), exactly lifts
an ordinary type-checked Go `Add(int64, int64) int64` body into language-neutral
entities and executes it through a canonical Seme-owned interpreter. Go is used
as the source oracle, but is absent from the runtime invocation. Unsupported
bodies reject explicitly; this is an independent execution proof, not a claim
of general Go execution. Run it with `./scripts/check-execution-v1.sh`.

[`Package Contract v1`](spec/package-contract-v1.md) adds the first useful
application boundary: an untouched ordinary Go quota policy with a stable typed
interface, explicit empty dependency/effect sets, a modular-integer runtime
assumption, and exact-fidelity evidence. Core Execution v2 runs it independently
and compares its behavior with Go. Run `./scripts/check-application-v1.sh`.

[`Target Contract v1`](spec/target-contract-v1.md) now discovers a real Go
standard-library logging dependency and `observability.log` effect from an
untouched package. [`Core Execution v3`](spec/core-execution-v3.md) lifts its
ordinary request and response structs as canonical records and field
expressions. The target contract emits an honest Wasm/Pulp plan: adapted host import and
typed boundary when policy permits, impossible when exact-only policy forbids
adaptation. Run `./scripts/check-target-v1.sh`. Wasm emission follows through
[`Wasm target v1`](spec/wasm-target-v1.md): a checked-plan backend emits a
deterministic 490-byte Pulp reactor and compares both Result variants and logging trace
with Go. `Pulp target v1` then runs that same reactor through actual Pulp and
proves repeated dynamic provider calls plus grant/denial behavior. The combined
milestone is [`Application Proof v1`](spec/application-proof-v1.md). Run
`./scripts/check-wasm-v1.sh` and `./scripts/check-pulp-v1.sh`, or show the whole
runtime slice with `./scripts/demo-application-v1.sh`.

[`Core Execution v4`](spec/core-execution-v4.md) is the next additive semantic
floor. It defines target-independent strings, byte sequences, and explicit
`Result<Ok, Error>` values while preserving frozen v2 and v3 byte-for-byte. The
Go target profile now lifts `string`, `[]byte`, and `(response, error)` into
those entities. [`Core Execution v5`](spec/core-execution-v5.md) adds the
minimum canonical conditional semantics needed to execute both bounded
Application Wire v2 Result variants. Provider conformance proves that local
names, comparison spelling, and keyed-field source order are not semantic
authority, while changed error text is preserved through Seme and Wasm.
[`Core Execution v6`](spec/core-execution-v6.md) adds canonical direct calls;
the Go helper and handler now lower to distinct Wasm functions.

Above frozen Kernel v1, Semantic Foundation v1 and Patch Module v1 define the
first versioned schema and transactional-edit contracts. Their checked module
artifacts and differential conformance gate are under `modules/`, `reference/`,
and `scripts/check-semantic-modules.sh`.
