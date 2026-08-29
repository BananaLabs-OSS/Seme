# Core Execution Semantics v1

Core Execution v1 is the first language-neutral executable semantic module.
It is intentionally distinct from the K0 module: Core Execution describes
program meaning, while K0 is one derived machine used to run an interpreter or
lowered artifact.

The module identity is `...9000`, initial revision `...9001`.

| Identity | Schema |
|---|---|
| `...9010` | IntegerType |
| `...9011` | Function |
| `...9012` | Parameter |
| `...9013` | ParameterRead |
| `...9014` | IntegerAdd |
| `...9015` | ExecutableProgram |

V1 defines fixed-width integers by width, signedness, and overflow policy.
Overflow value `0` means modular two's-complement wrapping. The first execution
profile accepts signed width 64 only.

A Function has ordered Parameters, a result IntegerType, and a body expression.
ParameterRead and IntegerAdd are the first expression forms. ExecutableProgram
selects an entry Function. These entities contain no Go syntax, Go AST nodes,
source positions, or Go runtime dependencies.

The first conformance proof lifts this exact ordinary Go subset:

```go
func Add(a int64, b int64) int64 { return a + b }
```

The lift is exact only when the parsed, type-checked body has one return of an
addition whose operands resolve to the two declared `int64` parameters. Every
other construct rejects as unsupported rather than receiving guessed meaning.

## Independent execution contract

The canonical interpreter is authored in S1 and checked into the repository as
canonical K0-module Seme plus its derived K0 image. Its runtime interface is:

```text
interpreter.k0 PROGRAM.seme ARGS.bin RESULT.bin
```

`ARGS.bin` is exactly two little-endian signed-64 bit patterns; `RESULT.bin` is
exactly one. The interpreter resolves the program entry, validates the signed
64-bit modular profile, follows stable references, and evaluates ParameterRead
and IntegerAdd. It rejects malformed inputs or unsupported canonical forms.

The conformance gate compares normal, negative, zero, and overflow results with
ordinary Go. Go constructs the canonical input and checks results, but the
runtime invocation contains no Go executable, source, AST, toolchain, or Go
runtime. Passing this gate means the scoped function runs through Seme. It does
not mean arbitrary Go programs run through Seme.
