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

That bounded application profile is complete at 36/36. The separate
[`Useful Project Bridge Profile v1`](spec/useful-project-bridge-profile-v1.md)
is now complete at 36/36 (100%): bounded Go, JavaScript, and Lua UPB-01
through UPB-12 are claimed. Each language is 12/12 (100%).
The profile measures complete
project snapshots, package and dependency closures, resources, configuration,
service boundaries, build placement, and project-scale reconciliation using
seven evidence classes. Its shape-only baseline is reported by
`./scripts/check-upb-v1-scorecard.sh`; a future score may advance only through
authoritative project gates. The shared UPB-12 gate projects one source-free
canonical project independently into ordinary Go, JavaScript, and Lua, executes
4,096 observations through all native projections, canonical Seme, standalone
Wasm, and pinned Pulp, and proves one convergent Project-v14 semantic edit.
Run `./scripts/check-upb12-shared.sh`. Downstream products, including
Workbench, remain
external consumers and cannot add product-specific behavior to Seme.

JavaScript UPB-01 lifts an ordinary multi-file ES-module project directly into
canonical Execution and neutral Project authority. Project v2 binds a complete
deterministic source inventory covering tracked, ignored, generated, vendored,
and opaque units. Canonical projection publishes a fresh native project,
preserves all nonsemantic bytes, passes Node behavior checks, and re-lifts to
identical Execution and Project bytes. Run
`./scripts/check-javascript-upb-01.sh`.

JavaScript UPB-02 preserves two real ES modules, public and package-private
declarations, exact relative import bindings, canonical signatures, and source
origins in Package v2 and Project v3. Canonical projection reconstructs the
native module boundary and re-lifts to identical meaning; native, canonical,
Wasm, and pinned Pulp results agree. Run
`./scripts/check-javascript-upb-02.sh`.

JavaScript UPB-03 resolves a local ES-module dependency and an exact pinned npm
package from independently hashed offline trees. Dependency v1 and Project v4
bind the lock selection, SHA-512 integrity, source, applicability metadata, and
content digest. Projection preserves native package metadata and repeats the
resolution without network access or lifecycle scripts. Run
`./scripts/check-javascript-upb-03.sh`.

JavaScript UPB-04 carries a typed `slice<i64>` and six `i64` values across a
real ES-module call boundary, then composes checked slice, fold, and map
operations. Package v2 preserves exact signatures and ownership; the complete
Project-v4 chain projects and re-lifts while native, canonical, standalone
Wasm, and pinned Pulp accept or reject every named vector identically. Run
`./scripts/check-javascript-upb-04.sh`.

JavaScript UPB-05 advances the same native project to Execution v36,
authenticated structural records, generated generic-result ownership, typed
defaults and validation, and three ordered lifecycle stages. Provider-neutral
assembly produces reproducible Configuration v3 and Project v8 authorities.
Native JavaScript, canonical Seme, the standalone canonical Wasm cell, and
pinned Pulp agree, while projected ES modules re-lift exactly. Run
`./scripts/check-javascript-upb-05.sh`.

JavaScript UPB-06 binds exact text and opaque binary project resources through
Resource v1 and Project v9. Source discovery remains provider-owned while
paths, ownership, digests, detached bytes, placements, validation, and project
composition remain language-neutral. Projection preserves both byte streams
and re-lifts the native modules exactly. Run
`./scripts/check-javascript-upb-06.sh`.

JavaScript UPB-07 resolves native JSDoc state records, validators, and one
v1-to-v2 migration into the neutral Durable State v1 contract. Project v10
binds the versioned family, canonical codec, bounds, string-keyed opaque-token
Load/CAS port, and an intentionally empty source-presentation manifest to the
exact Project-v9 graph. Native JavaScript, canonical Seme, standalone Wasm,
and pinned Pulp execute the pure migration identically; the separate neutral
host port suite proves atomic storage mechanics without claiming a Pulp or
database realization. Run `./scripts/check-javascript-upb-07.sh`.

JavaScript UPB-08 adds native command/event records and pure dispatch/replay
functions to the cumulative project. The project-neutral adapter resolves them
into Ordered Transport v1, and Project v11 binds one bounded ordered stream to
the exact Project-v10 graph. Native JavaScript, canonical Seme, standalone
Wasm, and pinned Pulp dispatch identically; separate host tests prove framing
and placement while refusing to mislabel Pulp's synchronous carrier as an
authoritative stream provider. Run `./scripts/check-javascript-upb-08.sh`.

JavaScript UPB-09 adds explicit clock, seeded-random, state/command/result, and
effect-bearing dispatch declarations. The native project graph assigns the
lifted `observability.log` effect to its exact package, while Controlled Effects
v1 and Project v12 remain language-neutral. Native JavaScript, canonical Seme,
standalone Wasm, and pinned Pulp agree on both value and logical effect trace;
live clock sampling and delivery remain explicit host boundaries. Run
`./scripts/check-javascript-upb-09.sh`.

JavaScript UPB-10 derives a complete target closure from the authenticated
Project-v12 graph and binds its executable Target-v1 plan through Project v13.
All 35 requirements resolve explicitly: 31 exact and four native islands for
live clock sampling, external effect delivery, durable storage, and ordered
transport. Two source-free deployments reproduce byte-for-byte; exact-only
policy, plan tampering, and existing destinations fail closed. Provider rules
use a JavaScript-owned realization namespace while the contracts and project
graph remain language-neutral. Run `./scripts/check-javascript-upb-10.sh`.

JavaScript UPB-11 adds monotonic complete-project snapshots with last-valid
retention and one identity-bound semantic rename across an exported declaration,
named import, and structured configuration reference. Native JavaScript is
projected atomically, validated, and re-imported under the unchanged canonical
identity. Both complete Project-v12 revisions are independently placed, and
Project v14 binds their exact Project-v13 authorities through Patch v1 and a
native-validation transcript. Stale native bytes/revisions, simultaneous
native and semantic changes, ambiguity, collisions, malformed identifiers,
tampered metadata, and existing destinations fail closed. Run
`./scripts/check-javascript-upb-11.sh`.

Lua UPB-01 discovers the complete ordinary two-file cumulative Lua application
and classifies tracked, ignored, generated, vendored, and opaque files under a
deterministic source snapshot. A Lua-owned manifest adapter derives the entry
signature and exact owned effect from canonical meaning; neutral Project v1 and
v2 bind the package and source inventory. Canonical projection publishes a
fresh native Lua project, preserves every nonsemantic byte contract, executes
identically under Neovim's Lua runtime, and re-lifts to identical canonical
bytes. Digest drift, symlinks, malformed graphs, denied capabilities, and
publication collisions reject atomically. Run `./scripts/check-lua-upb-01.sh`.

Lua UPB-02 imports ordinary static `require` modules without placing Lua module
mechanics in Core. Package v2 retains module ownership, returned-table exports,
private members, import bindings, source locations, and canonical signatures;
Project v3 authenticates the graph. Projection recreates the native module tree
and exact canonical re-lift, while native Lua, canonical, Wasm, and pinned Pulp
observations agree. Run `./scripts/check-lua-upb-02.sh`.

Lua UPB-03 resolves a bounded static rockspec plus integrity-pinned lock into
the neutral Dependency-v1 closure and authenticates it with Project v4. The
local module and external LuaRocks-style tree resolve repeatedly offline;
projection preserves the native rockspec and lock exactly. Floating versions,
substitution, unknown lock fields, unsafe paths, digest changes, malformed
authority, and publication collisions reject. Run `./scripts/check-lua-upb-03.sh`.

Lua UPB-04 passes a typed `slice<i64>` from the root module through a genuine
`require` binding into a separately owned package function. Its canonical call
and Package-v2 signatures retain the aggregate boundary without flattening the
modules or translating through another language. The authenticated dependency
chain, native/projected Lua, canonical evaluator, Wasm, and pinned Pulp agree;
typed boundary failures reject. Run `./scripts/check-lua-upb-04.sh`.

Lua UPB-05 binds typed defaults, validation, and the deterministic
configuration-to-policy-to-application initialization graph through neutral
Configuration v3 and Project v8. Lua record annotations are owned by the
package graph and survive modular projection. Native Lua, canonical, canonical
Wasm VM, and pinned Pulp agree; ambient environment access and invalid
lifecycle dependencies reject. Run `./scripts/check-lua-upb-05.sh`.

Lua UPB-06 binds one text and one binary resource by stable identity, size,
media type, destination, and SHA-256 through Resource v1 and Project v9.
Detached blobs and projected native resource bytes reproduce exactly. Changed
content, traversal, wrong digest or size, duplicate destinations, unknown
fields, symlinks, stale configured authority, and collisions reject. Run
`./scripts/check-lua-upb-06.sh`.

Go UPB-03 adds a resolved, integrity-pinned local-plus-ecosystem dependency
closure, authenticates it with Project v4, and proves that it applies to the
exact canonical package edge and inventoried `go.mod`. Resolution repeats
offline; native, canonical, standalone Wasm, pinned Pulp, and strict projection
evidence agree. The ecosystem dependency is intentionally metadata-only in
this cell. UPB-04 covers typed local package boundaries; it does not by itself
claim calls into arbitrary ecosystem packages. Run
`./scripts/check-go-upb-03.sh`.

Go UPB-04 adds complete typed declaration ownership and authenticated local
cross-package calls without flattening the three-package project. The full
UAB-11 application passes 2,048 canonical, standalone-Wasm, and pinned-Pulp
observations; Package v3 projection rebuilds natively and re-lifts to exact
canonical meaning. Arbitrary ecosystem-package calls remain outside this
bounded claim. Run `./scripts/check-go-upb-04.sh`.

Go UPB-05 adds explicit typed configuration and ordered initialization without
ambient environment state. One Execution-v36 run binds the source inventory,
dependency closure, Package-v4 ownership, executable program, Configuration-v3
plan, and Project-v8 snapshot. Three initializers execute through the
authenticated configuration executor; the cumulative 2,058-case corpus agrees
across native Go, canonical execution, standalone Wasm, and pinned Pulp.
Authenticated ordinary-Go projection passes its native tests and re-lifts to
the exact v36 execution graph. Run `./scripts/check-go-upb-05.sh`.

Go UPB-06 adds strict project-owned resources without adding filesystem
semantics to Core. Resource v1 authenticates detached UTF-8 and binary blobs;
Project v9 binds that manifest to the cumulative configured project. All 2,066
native/canonical/Wasm/Pulp observations agree, authenticated projection copies
resource bytes exactly, and hostile paths, digests, symlinks, limits, bundle
mixing, and output collisions reject atomically. Run
`./scripts/check-go-upb-06.sh`.

Go UPB-07 adds a neutral versioned durable-state contract with one v1-to-v2
migration and an authenticated Load/compare-exchange host boundary. A
2,080-case pure planner agrees across native Go, canonical execution,
standalone Wasm, and pinned Pulp; the stateful port is deliberately host-only
because pinned Pulp has no matching opaque-token CAS provider. Source
Presentation v1 preserves native Go aliases without turning them into Core
types. The source-free report, deterministic 14-artifact bundle, ordinary-Go
projection, semantic re-lift, source-bound revision changes, fixed point, and
atomic adversaries pass. Run `./scripts/check-go-upb-07.sh`.

Go UPB-08 adds a neutral ordered request/response transport contract with
bounded whole-frame authority, sequencing, correlation, retained replay, and
capability-authenticated host placement. A deterministic 4,096-observation
corpus agrees across native Go, canonical Seme, standalone Wasm, and pinned
Pulp. The authoritative stream port separately proves exact bytes, strict
framing under fragmentation/coalescing, retry after post-commit send failure,
and rejection by pinned Pulp, whose synchronous opaque call carrier is not a
TransportPort. Deterministic 16-artifact bundles, source-free reports, native
projection, exact re-lift, fixed point, and atomic adversaries pass. Run
`./scripts/check-go-upb-08.sh`.

Go UPB-09 adds neutral declared clock inputs, a versioned seeded random
transition, one ordered external Boolean effect request, and authenticated
replay. A deterministic 4,096-observation corpus agrees across native Go,
canonical Seme, standalone Wasm, and pinned Pulp. Live clock sampling and
effect delivery remain an explicit authenticated Go host boundary; Pulp carries
the pure planner/replay and ambient time or entropy cannot substitute for the
declared providers. Deterministic bundles, real replay/tamper evidence, and a
source-free ordinary-Go projection/re-lift semantic fixed point pass. Run
`./scripts/check-go-upb-09.sh`.

Go UPB-10 derives 92 target requirements from the authenticated Project-v12
closure and resolves 88 exactly while retaining clock delivery, external
effects, DurablePort, and TransportPort as four explicit Go-host native
islands. Project v13 binds the executable Target-v1 plan. A closed deployment
also authenticates its provider catalog, canonical VM Wasm, pinned Pulp cell
manifest, typed native-island launch manifest, and source-free report. Two
independent deployments match byte-for-byte; exact-only policy produces four
impossible diagnostics and publishes nothing. Run
`./scripts/check-go-upb-10.sh`.

Go UPB-11 performs an identity-bound exported-function rename across two
packages and 23 typed occurrences. It atomically projects a new ordinary-Go
project, runs its tests offline, re-lifts it to the independently expected
canonical graph, preserves the declaration's Seme identity, and binds the
prior and resulting Project-v13 authorities with Patch v1 in compact Project
v14. Two complete finalizations match byte-for-byte; stale, malformed,
conflicting, test-failing, tampered, symlink, and output-collision adversaries
publish nothing. Run `./scripts/check-go-upb-11.sh`.

The first bounded foundation is now executable: [`Go Project Build v1`](spec/go-project-build-v1.md)
lifts a native three-package Go fixture, compiles its canonical G1 with the
frozen Seme compiler, validates Execution/Package/Project instances, and emits
one content-derived `.seme` Project artifact. Client revision changes and
comment/filename-only edits reproduce byte-identical artifacts; a dependency
behavior edit changes them. Run `./scripts/check-go-project-build-v1.sh`. This
is supporting evidence, not a completed seven-class UPB cell.

Project Contract v2 adds a separate canonical SourceInventory bound to the
semantic ProjectSnapshot. Go UPB-01 discovers every file into
tracked, ignored, generated, vendored, or opaque classes; rejects unsafe paths
and drift; emits and validates the inventory; captures detached digest-addressed
bytes; preserves nonsemantic regions; and repeats the existing native,
canonical, Wasm, and Pulp behavior proof. Run
`./scripts/check-go-upb-01.sh`. Its canonical multi-package projector preserves
package ownership and duplicate names, regenerates tracked Go, restores all
other regions from the detached bundle, passes native tests, and re-lifts
byte-identically. Checked fidelity and atomic adversary evidence complete all
seven evidence classes for this bounded cell.

UAB-01 is complete in all three language columns: native multi-source packages,
typed named functions, parameters, and calls lift directly to canonical Seme,
project back to their own ecosystem, re-lift identically, and agree with the
claimed targets. Run `./scripts/check-uab-v1-go-01.sh`,
`./scripts/check-javascript-uab-01.sh`, and
`./scripts/check-lua-provider-v1.sh`. The machine-readable profile score is
reported by `./scripts/check-uab-v1-scorecard.sh`.

UAB-02 is complete in all three language columns for the frozen value
vocabulary. Shared scalar, collection, and tagged-composite observations agree
across original and projected native source, independent structural canonical
evaluation, standalone Wasm, and pinned Pulp, with representation-appropriate
rejection evidence. Run `./scripts/check-uab-v1-go-02.sh`,
`./scripts/check-javascript-uab-02.sh`, and `./scripts/check-lua-uab-02.sh`.

UAB-03 is complete in all three columns for bounded compositional control flow:
locals, mutable assignment, modular i64 arithmetic, comparison, observable
short-circuit logic, conditionals, loops, and early return. Each language uses
one shared corpus across original and projected native source, canonical
evaluation, standalone Wasm, and pinned Pulp. Run `./scripts/check-go-uab-03.sh`,
`./scripts/check-javascript-uab-03.sh`, and `./scripts/check-lua-uab-03.sh`.

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

[`Core Execution v31`](spec/core-execution-v31.md) adds neutral explicit
`OptionType`, `OptionNone`, and `OptionSome` values. [`Core Execution
v32`](spec/core-execution-v32.md) adds typed variant bindings, total
Result/Option matching, and exact byte literals/equality. Both are additive;
neither imports a source language's null, tuple, exception, or object model.
Run `./scripts/check-execution-v31.sh` and `./scripts/check-execution-v32.sh`.

[`Core Execution v37`](spec/core-execution-v37.md) adds neutral `UnitType` and
`UnitValue`; [`v38`](spec/core-execution-v38.md) adds ordered expression
evaluation with a discarded result. [`v39`](spec/core-execution-v39.md) adds a
typed native-invocation boundary so canonical surrounding logic can retain an
exact language/runtime callable without claiming that mechanic is portable.
Targets must supply the named realization or reject it. Run
`./scripts/check-execution-v37.sh`, `./scripts/check-execution-v38.sh`, and
`./scripts/check-execution-v39.sh`.

[`Core Execution v40`](spec/core-execution-v40.md) adds a typed native value
method boundary with an explicit receiver, ordered arguments, native signature,
and canonical result type. The Go provider evaluates the receiver once and
retains only non-variadic value methods whose result is representable; pointer
mutation and dynamic dispatch remain native islands rather than receiving a
false portability claim. The same provider milestone generalizes its existing
slice semantics across `int64`, `bool`, and `string` without changing the
established `[]int64` identity, and lifts single-binding Go `if` initializers
with their true lexical scope. Run `./scripts/check-execution-v40.sh`.

[`Core Execution v41`](spec/core-execution-v41.md) adds language-neutral
ordered products and statically typed projection. The Go provider uses them for
bounded multiple-result functions, local destructuring, and forwarded native
value-plus-error calls while evaluating each producer once. Runtime-owned
types are explicit: Go `error` is retained as a typed Go-native value, and its
nil test is an attributable native operation rather than a fabricated
universal exception rule. Map comma-ok remains the existing neutral Option
mapping. Run `./scripts/check-execution-v41.sh`.

[`Core Execution v42`](spec/core-execution-v42.md) generalizes the existing
neutral `NativeType` boundary across function parameters and results. The Go
provider retains imported/runtime-owned types by fully qualified `go/types`
spelling while continuing to reject unsupported local semantic records. This
adds no schema and does not make native fields, operators, memory, or methods
portable. Run `./scripts/check-execution-v42.sh`.

[`Core Execution v43`](spec/core-execution-v43.md) adds typed native field
observation with an explicit language, qualified field identity, receiver, and
canonical result type. The Go provider retains exported observations on
runtime-owned values without reclassifying unsupported local records. It also
retains typed native binding reads, Unit-returning procedures, and native items
inside ordered product results. Run
`./scripts/check-execution-v43.sh`.

[`Core Execution v44`](spec/core-execution-v44.md) completes the first typed
native-value flow through Go local bindings. Ordered native call products can
be projected once into locals, and Go-owned pointer or interface methods remain
explicit native invocations rather than becoming false neutral semantics. Run
`./scripts/check-execution-v44.sh`.

[`Core Execution v45`](spec/core-execution-v45.md) adds an attributable native
default-value operation. The Go provider can lift ordinary local `var`
declarations while keeping Go-specific zero initialization explicit; neutral
types continue using neutral zero constructors. Run
`./scripts/check-execution-v45.sh`.

[`Core Execution v46`](spec/core-execution-v46.md) makes typed native result
flow symmetric: single results and ordered product items from native functions
or methods can enter ordinary local bindings without losing their native type
identity. Run `./scripts/check-execution-v46.sh`.

[`Core Execution v47`](spec/core-execution-v47.md) preserves runtime-owned
intermediate field values as explicit native types, so nested observations such
as Go's `request.URL.Path` remain attributable and can return neutral values to
canonical Seme. Run `./scripts/check-execution-v47.sh`.

[`Core Execution v48`](spec/core-execution-v48.md) permits ordinary function
products to mix neutral values with explicitly Go-owned result types while
keeping application-owned records out of the runtime boundary. Run
`./scripts/check-execution-v48.sh`.

[`Core Execution v49`](spec/core-execution-v49.md) retains Go-owned fields
inside canonical application records and evaluates multi-result `if`
initializers once into branch-scoped typed products. Run
`./scripts/check-execution-v49.sh`.

[`Core Execution v50`](spec/core-execution-v50.md) evaluates multi-result
assignment initializers once and threads fallthrough continuations through
nested early-return branches. Run `./scripts/check-execution-v50.sh`.

[`Core Execution v51`](spec/core-execution-v51.md) keeps unsupported
application-owned Go values as explicit typed native boundaries, including
receiver fields and comma-ok type assertions, while preserving canonical
surrounding control flow. Run `./scripts/check-execution-v51.sh`.

[`Core Execution v52`](spec/core-execution-v52.md) represents single blank
assignments as exactly-once discarded evaluations instead of malformed local
declarations. Run `./scripts/check-execution-v52.sh`.

The bounded [`Pure Value ABI v1`](spec/pure-value-abi-v1.md) derives recursive
Bytes, Result, and Option layouts from canonical types. Its first executable
reactor profile proves `Option<Result<bytes,text>> -> Boolean` with nested
matches, UTF-8 validation, deterministic Wasm, malformed-request rejection,
and pinned Pulp execution. It is deliberately not described as general
composite lowering. Run `./scripts/check-composite-runtime-v32.sh`.

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
parses and type-checks complete in-memory module snapshots, including a bounded
closure of sibling local packages. It compositionally lifts supported
functions, methods, locals, and declared effects, rejects stale revisions, and
executes the resulting canonical graph. Valid results include typed reference
occurrences bound to stable semantic declaration identities, allowing editors
to navigate without name-based inference. This remains a bounded source closure,
not general dependency or build-variant support. Run
`./scripts/check-language-service-v1.sh` and
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
