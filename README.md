# Seme

Seme is a canonical semantic programming platform. Syntax is a projection,
mechanics are composable semantic modules, packages and runtimes are explicit
dependencies, and execution targets declare how faithfully they realize those
semantics.

Seme is target-independent. WebAssembly is the first-class execution target and
primary portability proof, not the semantic foundation or the only backend.

## Try the usable Go profile

On Linux amd64 with Go and Node installed:

```sh
./seme doctor
./seme demo
```

Against an ordinary project matching the documented profile:

```sh
./seme build PATH/TO/PROJECT
./seme run PATH/TO/PROJECT 40 2 50 tenant-a
./seme audit PATH/TO/PROJECTS
```

The command runs the native tests, imports and validates canonical semantics,
reports fidelity, emits Wasm, and preserves inspectable evidence under
`PROJECT/.seme/`. It does not rewrite the Go sources. See
[`Seme CLI v1`](spec/cli-v1.md) for the exact supported surface and outputs.

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
deterministic 493-byte Pulp reactor and compares both Result variants and logging trace
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
[`Core Execution v7`](spec/core-execution-v7.md) adds typed integer literals;
the proof executes a source-derived nested `+ 0` through Seme and Wasm.
Provider revisions cover nested module sources, and target analysis resolves
the real `Admit -> internal/policy.WithinLimit` imported source call through the
same canonical call and Wasm layers.
Equivalent helper comparison direction is normalized, while resolved
parameter identities flow through canonical reads into generated Wasm locals.
The provider performs that work through a shared typed expression analyzer
rather than separate AST recognizers for each proof command. Canonical emission
walks the analyzed tree while preserving the existing frozen expression IDs.
Wasm lowering now walks the resulting integer graph recursively with cycle and
size guards instead of embedding the helper's instruction sequence.
Imported package functions are also first-class provider declarations with
stable package-qualified identities.

[`Core Execution v8`](spec/core-execution-v8.md) begins the compositional
statement layer with provider-neutral ordered blocks and returns. Its bounded
Go proof lifts a typed function as `Function -> Block -> Return -> expression`
without selecting behavior by application or function name, while regenerating
the frozen v2-v7 modules byte-for-byte. Run
`./scripts/check-execution-v8.sh`.

[`Core Execution v9`](spec/core-execution-v9.md) adds compositional signed-i64
multiplication, and [`Core Execution v10`](spec/core-execution-v10.md) adds
ordered subtraction. Their bounded proofs cover recursive analysis, canonical
emission, modular native parity, and guarded Wasm instruction lowering. They do
not yet claim that the legacy complete Wasm application builder can execute a
v8 structured function. Run `./scripts/check-execution-v9.sh` and
`./scripts/check-execution-v10.sh`.

[`Core Execution v11`](spec/core-execution-v11.md) adds canonical Boolean
literals and left-to-right short-circuit conjunction. The Go provider,
semantic evaluator, and recursive Wasm instruction lowerer compose Boolean
operations with integer comparisons; malformed canonical Boolean values reject.
The complete structured-function Wasm application boundary remains tracked as
unfinished. Run `./scripts/check-execution-v11.sh`.

[`Core Execution v12`](spec/core-execution-v12.md) closes that bounded runtime
gap with a canonical-graph-driven pure-function ABI. Parameter and result
layouts are derived from canonical i64/Boolean types; the same backend executes
different signatures and behaviors as standalone Wasm and through Pulp, with
exact-length and canonical-Boolean rejection. Run
`./scripts/check-execution-v12.sh`.

[`Core Execution v13`](spec/core-execution-v13.md) adds explicit total-return
branching, nested blocks and multiple return sites, short-circuit Boolean OR,
and bounded exact UTF-8 literal/concatenation/equality expressions. The Go
provider normalizes fallthrough returns into explicit branches, and the same
canonical tree runs as standalone Wasm and through Pulp. Run
`./scripts/check-execution-v13.sh`.

[`Core Execution v14`](spec/core-execution-v14.md) adds the variable-width
`seme.pure-abi/v2` realization for exact UTF-8 string parameters and results.
Ordered descriptors, strict validation, runtime concatenation/equality, bounded
allocation, standalone Wasm, and capability-free Pulp execution are checked by
`./scripts/check-execution-v14.sh`; scalar ABI v1 remains frozen.

[`JavaScript Provider v1`](spec/javascript-provider-v1.md) is the first direct
non-Go source lift. A deliberately bounded, explicitly typed native JavaScript
function and its idiomatic Go equivalent compile to byte-identical canonical
Seme, then the JavaScript-derived graph executes through ABI v2 Wasm. The same
graph projects back to native JavaScript and re-lifts without semantic drift.
Ambiguous or coercive JavaScript rejects instead of being guessed. Run
`./scripts/check-javascript-provider-v1.sh`.

[`Core Execution v15`](spec/core-execution-v15.md) adds immutable lexical local
bindings, ordered binding statements, and local reads. Native Go `:=` and
JavaScript `const` views converge on the same canonical graph; JavaScript
projection re-lifts without drift, while integer and UTF-8 local programs run
standalone and through the applicable Pulp/Wasm path. Run
`./scripts/check-execution-v15.sh`.

[`Core Execution v16`](spec/core-execution-v16.md) turns isolated function
proofs into closed multi-function programs. Ordinary Go package calls and
native JavaScript module calls converge on explicit canonical `FunctionCall`
semantics; JavaScript projection re-lifts without drift. The bounded pure Wasm
target certifies membership and an acyclic call graph, then realizes calls by
safe target-side inlining before standalone and Pulp execution. Run
`./scripts/check-execution-v16.sh`.

[`Core Execution v17`](spec/core-execution-v17.md) restores the existing
record vocabulary to the generic pipeline. Go structs/composite literals/field
reads and JavaScript typedefs/plain objects/property reads converge on the same
canonical records, project without drift, and execute as internal immutable
values through Wasm and Pulp. Run `./scripts/check-execution-v17.sh`.

[`Core Execution v18`](spec/core-execution-v18.md) adds typed mutable places,
ordered assignment, and structured `While`. Native Go condition loops and
JavaScript `while` converge on one canonical graph and project without drift;
the bounded string/Boolean profile executes as real Wasm locals and loops
standalone and through Pulp. Run `./scripts/check-execution-v18.sh`.

[`Core Execution v19`](spec/core-execution-v19.md) adds provider-neutral
one-sided structured choice and realizes signed-i64 mutable places. Ordinary Go
and JavaScript `if` statements converge on `When`; both outcomes execute with
exact modular-i64 values standalone and through Pulp. Run
`./scripts/check-execution-v19.sh`.

The durable status, verified evidence, honest limits, environment state, and
recommended resumption point through v19 are recorded in the
[`Core v19 development checkpoint`](spec/development-checkpoint-v19.md).

[`Core Execution v20`](spec/core-execution-v20.md) adds ordered Foundation
effect invocation without placing capabilities or host APIs in Core. Bounded
Go logging and JavaScript console adapters converge on one canonical effect;
the generated Wasm records adapted fidelity and the required capability,
executes ordered observations through Pulp when granted, and traps without an
effect when denied. Run `./scripts/check-execution-v20.sh`.

[`Core Execution v21`](spec/core-execution-v21.md) adds language-neutral fixed
array construction and indexed reads. Idiomatic Go and JavaScript converge on
one canonical graph; valid boundary indexes agree natively and through Wasm
and Pulp, while negative and upper-bound reads trap in the certified target.
Fixed i64 array parameters also cross a canonical packed Wasm ABI derived from
their element type and length. Run `./scripts/check-execution-v21.sh` and
`./scripts/check-execution-v21-boundary.sh`.

[`Core Execution v22`](spec/core-execution-v22.md) adds typed lexical iteration
bindings and deterministic folds. Idiomatic Go `range` accumulation and
JavaScript `reduce` converge on one canonical program; empty, signed, and
overflow cases agree natively, in Wasm, and through Pulp. Run
`./scripts/check-execution-v22.sh`.

[`Core Execution v23`](spec/core-execution-v23.md) separates runtime-sized
slices from fixed arrays and source runtime mechanics. Go `[]int64` and
JavaScript `bigint[]` converge on one canonical slice fold; a bounded
descriptor/payload ABI executes changing lengths through Wasm and Pulp. Run
`./scripts/check-execution-v23.sh`.

[`Core Execution v24`](spec/core-execution-v24.md) adds neutral collection
length and runtime-checked computed indexing. Go and JavaScript empty-safe last
element access converge canonically and execute with negative/upper-bound traps
through Wasm and Pulp. Run `./scripts/check-execution-v24.sh`. The distinction
between finite roadmap completion and broad language readiness is maintained in
the [`progress dashboard`](spec/progress-dashboard.md).

The frozen [`Useful Application Bridge Profile v1`](spec/useful-application-bridge-profile-v1.md)
defines the next honest 100% milestone for Go, JavaScript, and Lua: 12 composed
application capabilities per language, five evidence classes per capability,
and one shared cumulative application. Broad language, runtime, standard
library, tooling, and third-party package compatibility remain separately
versioned conformance tracks.

[`Core Execution v25`](spec/core-execution-v25.md) adds neutral immutable
collection append and update with dynamically sized slice results. Native Go
and JavaScript forms converge canonically; projection remains native, and
alias-safe bounded results agree through standalone Wasm and Pulp. Run
`./scripts/check-execution-v25.sh`.

[`Core Execution v26`](spec/core-execution-v26.md) adds receiver-bound methods
and explicit updated-state-plus-result transitions. Native Go value receivers
and JavaScript class methods converge on byte-identical canonical meaning and
execute through Wasm and Pulp without mutating the receiver. Run
`./scripts/check-execution-v26.sh`.

[`Core Execution v27`](spec/core-execution-v27.md) adds neutral interfaces,
method requirements, explicit satisfaction witnesses, interface values, and
bounded dynamic calls. Native Go interfaces and JavaScript structural classes
converge on byte-identical canonical meaning; witness-certified offset and
scale implementations execute through deterministic standalone Wasm and
pinned Pulp while malformed signatures and unknown tags reject. Run
`./scripts/check-execution-v27.sh`.

[`Core Execution v28`](spec/core-execution-v28.md) adds typed function values,
explicit immutable captures, closure construction, capture reads, and indirect
calls. Native Go and JavaScript closures converge canonically; escaped and
independent environments execute through deterministic Wasm and pinned Pulp,
including a portable returned-closure ABI. Run
`./scripts/check-execution-v28.sh`.

[`Core Execution v29`](spec/core-execution-v29.md) adds explicit mutable
captures, ordered capture updates, mutable closure construction, and stateful
indirect calls. Native Go and JavaScript closure mutation becomes explicit
state-transition threading in canonical Seme; repeated calls and independent
environments agree through deterministic Wasm and pinned Pulp. Run
`./scripts/check-execution-v29.sh`.

[`Core Execution v30`](spec/core-execution-v30.md) adds neutral runtime-keyed
maps with empty construction, zero-on-missing lookup, and immutable update.
Native Go maps and JavaScript `Map` values converge on a generic map-valued
slice fold; repeated, negative, missing, reordered, empty, and bounded inputs
agree through deterministic Wasm and pinned Pulp. Run
`./scripts/check-execution-v30.sh`.

The [`Core v30 cumulative composition proofs`](spec/core-cumulative-proof-a.md)
close the separate architectural gate: one ordinary program combines runtime
collection traversal, calls, locals, conditional method dispatch, and explicit
state transition; a second combines runtime text with collection behavior.
Go and projected JavaScript re-lift to identical canonical Seme, and generic
lowering agrees through deterministic Wasm and pinned Pulp. Run
`./scripts/check-core-cumulative-v30.sh`.

[`Live Language Service v1`](spec/live-language-service-v1.md) defines the
provider-neutral incremental editing boundary: document identity, monotonic
client revisions, content digests, diagnostics, semantic source mappings, and
retention of the last valid canonical revision while an edit is incomplete.
The bounded [`Go incremental session v1`](spec/go-incremental-session-v1.md)
parses and type-checks complete in-memory package snapshots, compositionally
lifts supported functions, rejects stale revisions, and executes the resulting
canonical graph. Run `./scripts/check-language-service-v1.sh` and
`./scripts/check-go-session-v1.sh`.

[`Language service JSON-lines v1`](spec/language-service-jsonl-v1.md) exposes
that incremental session as a deterministic, bounded, zero-write process with
multiple isolated sessions and correlated `initialize`, `update`, and
`snapshot` requests. Artifact publication remains a separate authority. Run
`./scripts/check-language-service-jsonl-v1.sh`.

[`Certified canonical build v1`](scripts/CANONICAL-BUILD-V1.md) accepts pinned
canonical bytes and emits a certified Wasm artifact and bound ABI evidence
without accepting a source tree or provider input. Run
`./scripts/check-certified-canonical-v1.sh`.

The planned cross-language architecture and its fidelity rules are recorded in
the [`language bridge roadmap`](spec/language-bridge-roadmap.md). The generic
capabilities still to restore above the clean v7 baseline are tracked without
claim inflation in the
[`Core semantic restoration ledger`](spec/core-restoration-ledger.md).

Above frozen Kernel v1, Semantic Foundation v1 and Patch Module v1 define the
first versioned schema and transactional-edit contracts. Their checked module
artifacts and differential conformance gate are under `modules/`, `reference/`,
and `scripts/check-semantic-modules.sh`.
