# Universal Software Commons

## Purpose

The foundational problem is not that programming remains engineering. The
problem is that existing software is artificially divided by language,
runtime, platform, package manager, editor, and discoverability.

A C programmer should not need to adopt Python, rewrite a mature Python
package, or abandon it merely because that implementation lives in another
ecosystem. Software should be difficult because its problem is difficult, not
because useful prior work is trapped behind ecosystem membership.

The intended user promise is:

> Find any software. Understand it your way. Use it from anywhere.

## Three foundations

The software commons has three distinct foundations:

1. **Seme defines meaning.** It gives software stable semantic identity,
   composable mechanics, explicit assumptions, projections, revisions, and
   target-fidelity contracts.
2. **The registry preserves continuity and discovery.** It groups a concept,
   its capabilities, revisions, variants, extensions, facades, implementations,
   provenance, and compatibility in one discoverable lineage.
3. **Pulp provides execution and composition.** It selects and supervises
   implementations, grants explicit capabilities, connects providers and
   consumers, and manages lifecycle across Wasm, native processes, language
   runtimes, containers, engines, and remote services.

These layers are independent. Seme does not own execution, Pulp does not own
program meaning, and the registry does not become the canonical source of
either.

## Semantic stenography and progressive explicitness

Seme is intended as semantic stenography. A developer states only the decisions
that matter at the current point and level. Declared mechanics and policies
supply omitted decisions through reproducible elaboration.

Seme supports seven intended levels of programming and language expression,
from concise intent to target-specific construction. The exact versioned
taxonomy belongs in a separate language contract. The invariant is more
important than the names of the levels:

> Any region may become more or less explicit without forcing the remainder of
> the program to move to the same level.

An application may combine Erlang-style supervision, goroutine-style
concurrency, borrowing, Python- or Lua-like presentation, C-like data layout,
and assembly-level target control. These are declared mechanics and
projections, not an inseparable bundle inherited from one historical language.

Every inferred decision must remain inspectable. Its mechanic or policy,
reason, dependencies, stability rule, fidelity, and local override path must be
available. Concision may omit decisions from the user's immediate notation; it
must not conceal them permanently.

## Personalized languages without private dialects

Textual, visual, conversational, editor-native, and direct semantic interfaces
are projections of shared semantic identities. Neovim, a purpose-built IDE, a
game editor, a visual graph, and an AI agent may all edit the same program
through transactional semantic patches.

A personalized application language selects the projections and mechanics that
fit that application. It does not fork the underlying meaning. Another person
may inspect the same program through a different familiar projection or at a
different explicitness level.

No official IDE owns the program. Tooling integrates through protocols for
semantic queries, projection rendering, source-to-identity maps, patches,
elaboration inspection, execution, tracing, and diagnostics.

## Registry concept model

The registry is concept-oriented rather than a flat collection of unrelated
language packages.

```text
vector
├── 2d
│   ├── original implementation
│   └── alternate variant
├── 3d
│   └── extends or composes 2d
└── facades
    ├── 2d-only
    ├── 3d-only
    └── full
```

- A **package** is the overall evolving concept, such as `vector`.
- A **capability** or **subpackage** is a composable portion, such as `2d`.
- A **revision** continues one implementation's history.
- A **variant** is a different implementation of the same capability.
- A **facade** is a convenient composition of capabilities and selected
  variants.

A contributor may extend abandoned work without replacing its identity.
Competing implementations remain variants beneath the shared concept instead
of becoming disconnected `better-new-final` projects.

Registry records bind semantic contracts to implementation variants and record
runtime requirements, platform support, fidelity, capability requirements,
source, license, artifact digest, signatures, build provenance, conformance
evidence, advisories, and revocations. Content-addressed artifacts and
federation keep the registry mirrorable and prevent one service from owning the
commons.

## Existing software joins unchanged

Rewriting the existing ecosystem is not an admission requirement. A package
keeps its original source, build system, runtime, memory model, concurrency
model, and native tools.

Existing packages join progressively:

1. **Opaque package:** Pulp runs or reaches the unchanged implementation in its
   native environment. Seme makes no claim about its internals.
2. **Contract-wrapped package:** a semantic contract describes its public
   operations, values, effects, errors, guarantees, and runtime assumptions.
3. **Semantically mapped package:** important internal types, identities,
   mechanics, effects, and invariants are mapped with explicit fidelity.
4. **Native semantic package:** the implementation itself is canonical Seme and
   may be projected and lowered through compatible targets.

Only the second level is required for broad cross-ecosystem reuse. Deeper
mapping is optional and should be performed only when it provides value.

Original source remains readable through its native representation. A consumer
may inspect the semantic contract through a preferred projection without
pretending that an arbitrary Python, JVM, C, Erlang, or assembly implementation
has been losslessly translated into another language.

## Runtime islands and membranes

Languages assume architectures. Python packages may require Python objects,
reflection, exceptions, garbage collection, and C extensions. JVM packages may
require class loading, monitors, and JVM object identity. Erlang packages may
require BEAM processes, mailboxes, supervision, and selective receive. C and
assembly may require a specific ABI, pointer model, instruction set, or memory
ordering.

Seme does not erase these assumptions. Each native runtime remains a valid
execution island behind a typed membrane:

```text
Seme consumer
    -> semantic contract
    -> typed adapter
    -> Pulp capability and lifecycle boundary
    -> Python / JVM / BEAM / native / Wasm / engine / remote provider
```

Pulp starts or connects to the provider, grants only declared authority,
marshals boundary values, routes calls and events, supervises failure, applies
resource policy, and exposes operational cost. Hot loops and dense engine state
remain inside the appropriate runtime rather than crossing the membrane for
every operation.

## Boundary contract

A foreign-package contract must define at least:

- stable concept, capability, operation, and value identities;
- records, variants, optionals, lists, maps, text, bytes, and numeric behavior;
- opaque runtime-owned handles where serialization is inappropriate;
- synchronous calls, asynchronous calls, streams, callbacks, and events;
- deadlines, cancellation, retry and idempotency behavior;
- errors and failure translation;
- ownership and lifetime of buffers and handles;
- required effects and Pulp capabilities;
- runtime, ABI, platform, and target assumptions;
- fidelity and any impossible or emulated behavior;
- conformance evidence.

Cross-runtime calls have observable cost. Contracts and adapters should support
coarse operations, batching, streaming, and runtime-local handles instead of
encouraging millions of tiny serialized calls.

## Compatibility is evidence, not a name

Two variants beneath one capability are not automatically substitutable. They
may differ in ordering, precision, determinism, concurrency, failure behavior,
transactions, resource limits, security properties, or latency.

Consumers declare required guarantees. Providers declare supplied guarantees
and fidelity. Registry resolution selects a provider only when the contract is
satisfied. Shared conformance suites provide behavioral evidence for variants.
Compatibility is graded and explicit, never inferred merely from a shared
package name or familiar syntax.

## WebAssembly

WebAssembly is the first-class portable implementation artifact and a strong
Pulp cell boundary. It provides hashable binaries, linear-memory isolation,
portable execution, capability-shaped imports, and broad language tooling.

WebAssembly remains a target and package bridge, not canonical Seme meaning or
the only valid runtime. Native engines, language VMs, GPU execution, containers,
and remote providers remain legitimate when their mechanics or performance
requirements make them the faithful choice.

## First interoperability proof

The first complete proof should use one mature, unchanged package from another
ecosystem:

1. retain its original source and native build/runtime;
2. register it beneath one shared concept and capability;
3. describe several operations with a Seme semantic contract;
4. launch or connect to it through a Pulp adapter;
5. call it from consumers associated with two different ecosystems;
6. display its contract through two familiar projections;
7. provide a second implementation variant;
8. run one shared conformance suite against both variants;
9. resolve between them based on target, capability, fidelity, and trust;
10. measure and expose boundary cost.

Success proves the foundational claim:

> Existing software can join without being rewritten, and consumers can reuse
> it without adopting its implementation language.

## Non-goals

The commons does not eliminate requirements, architecture, performance work,
distributed-systems reasoning, security policy, governance, maintenance, or
other intrinsic engineering. It removes accidental barriers that force
engineers to rediscover or rewrite existing software before addressing those
real problems.
