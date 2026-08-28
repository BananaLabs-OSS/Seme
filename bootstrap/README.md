# Bootstrap

The bootstrap begins with `seme-seed-linux-amd64`, a frozen Linux-amd64 ELF
program implementing the Hex0 contract in `HEX0.md`. Its human-auditable source
is `seed-linux-amd64.S`.

The seed is intentionally much smaller than the Seme kernel. It knows nothing
about semantic modules, projections, packages, mechanics, types, or compiler
policy.

## Construction versus reproduction

During bring-up, a host assembler and linker may construct the frozen binary
from the assembly source. They are not intended to remain in the trusted path.
The checked-in hexadecimal image must reproduce the frozen binary through the
seed itself:

```text
seme-seed-linux-amd64 seed-linux-amd64.hex rebuilt-seed
compare seme-seed-linux-amd64 rebuilt-seed
```

That exact check passes for the checked-in Linux-amd64 seed. Its SHA-256 is
recorded in `seed-linux-amd64.sha256`. This completes the first independently
reproducible rung.

## A0 rung

`seme-a0-linux-amd64` implements the canonical byte and unsigned-integer
notation in `A0.md`. The Hex0 seed constructs it from `a0-linux-amd64.hex`; it
then reconstructs itself from `a0-linux-amd64.a0`. Two generated generations
are byte-identical and pass malformed-input and overflow checks.

A0 deliberately has no labels or semantic concepts. The next rung is a small
structured kernel notation, not a larger native assembler.

## K0 rung

`seme-k0-linux-amd64` implements the draft portable execution profile in
`K0.md`. Both Hex0 and A0 construction inputs reproduce the checked executable
byte-for-byte. P0 currently validates canonical images and executes unsigned
integer arithmetic, comparisons, locals, stack operations, control-flow loops,
and return.

The conformance suite proves two independent programs return `42`, executes a
looping sum, and rejects uninitialized local reads, invalid branch targets,
non-canonical ULEB values, and trailing bytes. P0 is not Kernel v1: complete
branch-witness validation, functions, byte values, effects, and structured
diagnostics remain required before the kernel compiler can be expressed.

P1 adds a bounded function table and isolated call frames. The current executor
passes zero- and two-parameter calls, preserves all P0 fixtures, and rejects
invalid call indices, entry parameters, and attempts to observe caller locals
from a callee. Function bodies are still validated lazily; complete preflight
validation remains a freeze gate.

P2 adds bounded byte buffers and explicit capability bits for arguments,
filesystem reads, and filesystem writes. A portable P2 program reads one path
and writes a byte-identical copy to another. Undeclared effects and byte
out-of-bounds access are rejected before the host operation is performed.

Every P1/P2 function is structurally scanned before entry execution. Unreachable
invalid opcodes, malformed operands, bad indices, undeclared effects, incorrect
code sizes, and mismatched branch witnesses are rejected before any effect can
run. Semantic control-flow validation remains intentionally above this native
bootstrap boundary.

The executor also provides unsigned division/remainder and complete 64-bit
bitwise/shift primitives. Together with byte buffers, these are sufficient for
a portable K0 program to parse decimal/hexadecimal source and emit canonical
integer encodings; future source readers no longer require new arithmetic in
assembly.

## First portable compiler

`a0-byte-compiler.k0` is the first compiler program executed by portable K0
rather than native assembly. Its checked A0 construction source is
`a0-byte-compiler.k0.a0`. It accepts the auditable `bHH` subset used by checked
bootstrap construction sources, ignores whitespace and `#` comments, and emits
the represented bytes through declared argument/read/write capabilities.

The portable compiler compiles `k0-linux-amd64.a0` into a byte-identical copy of
the frozen K0 executor. Hex0 construction, native A0 construction, and portable
K0 compilation therefore converge on the same executable bytes. Decimal A0 and
the future human Seme notation remain separate compiler milestones.

## Seed responsibilities

- accept exactly an input and output path;
- decode pairs of hexadecimal digits;
- ignore ASCII whitespace and `#` line comments;
- reject invalid or incomplete input;
- write output without invoking a shell or subprocess;
- return deterministic exit statuses.

It has no dynamic-library dependency and uses Linux syscalls directly.
