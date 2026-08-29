# Seme

Seme is a canonical semantic programming platform. Syntax is a projection,
mechanics are composable semantic modules, packages and runtimes are explicit
dependencies, and execution targets declare how faithfully they realize those
semantics.

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

- Kernel v1 is being specified; it is not frozen.
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
- Host assembly tools remain optional construction and cross-check conveniences.
- The S0 bootstrap compiler is self-reproducing; the semantic Kernel v1 compiler
  is not self-hosted yet.

See [`spec/architecture.md`](spec/architecture.md) and
[`bootstrap/README.md`](bootstrap/README.md).
