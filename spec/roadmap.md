# Product proof roadmap

Seme is developed through evidence-bearing layers:

```text
bootstrap and self-hosting
    -> trustworthy Kernel
    -> transactional semantic edits
    -> ordinary-project integration
    -> independent lowering
    -> package interoperability
    -> real Pulp workload
    -> WebAssembly portability
```

Completing a later layer does not waive an earlier layer's conformance gates.

## 1. Trustworthy Kernel v1 — complete

The structural Kernel v1 boundary is frozen. The executable evidence is tracked
in the [`Kernel v1 freeze evaluation`](kernel-v1-freeze-audit.md) and pinned in
[`KERNEL-V1-FREEZE.md`](../compiler/KERNEL-V1-FREEZE.md).

- Emit deterministic structured diagnostics with stable rule, revision, entity,
  field/list path, and structured argument identities.
- Complete schema/import, effects/capabilities, constraints/holes, provenance,
  refinement, unknown-field, and migration conformance.
- Freeze the canonical format, identities, validation behavior, and migrations.

Located Diagnostics 1 is the frozen structural validator: it preserves valid
inputs exactly, emits canonical reports for the invalid corpus, carries
trustworthy revision/entity identities, and uses the explicit unavailable
identity sentinel where malformed input cannot establish a field identity.
Richer semantic-module diagnostics remain versioned work above Kernel v1.

## 2. Transactional semantic editing

- Apply identity-based patches against an explicit starting revision.
- Commit the complete patch or leave canonical state unchanged.
- Preserve identity across rename, movement, formatting, and projection changes.
- Tombstone deleted identities and report conflicts deterministically.

## 3. Ordinary Go project proof

Status: complete for the explicitly scoped Provider Contract v1 Go profile.
The gate is `./scripts/check-provider-v1.sh`; broader Go constructs remain
separate provider-profile work rather than implied support.

The Go toolchain is an optional external ecosystem provider and is not part of
Seme's repository, trusted bootstrap, or independent reproduction path.
The first Go provider proves Provider Contract v1; it is not a claim of general
Go support. Later providers must reuse the same patch and reconciliation
contracts rather than introduce language concepts into the Kernel.

The milestone must demonstrate:

1. open an ordinary, non-Seme-authored Go repository;
2. import understood semantics and preserve unsupported source opaquely;
3. perform one identity-preserving semantic edit;
4. project the smallest valid change back into the ordinary repository;
5. preserve untouched bytes except where the native formatter contract requires
   a change;
6. pass the repository's original tests using its normal Go toolchain;
7. re-import the edited repository;
8. reconstruct the same stable semantic entities and relationships, with only
   the intentional semantic revision changed;
9. remove Seme and leave a normal usable Go repository.

Compilation success alone does not satisfy this milestone. The re-import check
is the semantic round-trip proof. Identity equality, relationship equality, and
unchanged-entity equality must be compared independently of source formatting.

## 4. Independent lowering

Complete for the frozen Core Execution v1 profile:

- the Go provider parses and type-checks the exact `int64` parameter-addition
  subset, then emits language-neutral canonical entities;
- the Seme-owned canonical interpreter executes those entities without Go at
  runtime and matches ordinary Go across normal, negative, zero, and signed
  overflow vectors;
- unsupported constructs reject explicitly instead of receiving guessed
  semantics.

This proves the independent-execution contract, not general Go lowering. The
next milestone broadens only as required by the package/application proof.

## 5. Package and application proof

Status: complete for Package Contract v1's scoped quota-policy profile.

- An ordinary Go package remains untouched and exposes a typed policy boundary.
- Canonical Package entities record the interface, explicitly empty dependency
  and effect sets, modular-integer runtime assumption, and exact fidelity with
  evidence.
- Core Execution v2 independently executes the package policy and matches Go,
  including signed overflow.

This establishes the package/application contract. The subsequent target/Wasm
proof exercises a nonempty dependency and effect; actual Pulp execution is the
remaining real-workload boundary.

## 6. WebAssembly portability proof

Status: canonical planning, artifact execution, and actual Pulp capability
enforcement complete for the scoped logging profile.

- Target Contract v1 now analyzes a real `go:log` dependency and
  `observability.log` effect.
- The Wasm/Pulp plan records an adapted host-import Boundary and remains
  executable only when adaptation policy permits it.
- Exact-only policy produces impossible resolutions rather than dishonest
  exact fidelity.
- A checked-plan backend emits a deterministic 490-byte Wasm reactor, executes it
  with the declared logging host import, and matches Go result/effect traces.
- Core Execution v3 canonically represents the ordinary Go request/response
  records, field reads, and response construction; the backend derives its
  codec layout from those entities.
- Core Execution v4 freezes canonical strings, bytes, and Result success/error
  values. Core Execution v5 and the Go provider lift the source guard and
  execute bounded ResultOk and ResultError paths through Application Wire v2.
- The bounded Go provider accepts semantically equivalent comparison operand
  order, local names, and keyed-record field order, and propagates changed
  source error literals through Seme into Wasm without backend templates.
- Core Execution v6 preserves the ordinary Go handler-to-helper call edge;
  the target validates both functions and emits a separate Wasm helper call.
- Go provider revisions cover recursive module sources, so imported-package
  edits cannot remain hidden behind a valid root-package manifest.
- The application profile now resolves and lifts the real
  `Admit -> internal/policy.WithinLimit` package edge from source, preserving it
  as two canonical and two Wasm functions.
- The helper expression profile resolves operand parameter identities, ignores
  presentation-only parentheses, and normalizes equivalent comparison
  direction; Wasm local reads are derived from the canonical graph rather than
  assumed parameter positions.
- A shared recursive Go expression analyzer now separates AST recognition from
  profile matching for ParameterRead, IntegerAdd, and IntegerLessEqual. New
  canonical expression forms extend this layer instead of adding another
  fixture-specific parser.
- Canonical helper-expression emission walks that analyzed tree recursively.
  Frozen v6 expression identities remain stable; additional nested nodes use
  deterministic normalized semantic paths rather than source offsets.
- Wasm helper lowering recursively compiles the canonical integer graph with a
  node budget and cycle detection. The module builder receives validated
  instructions instead of embedding the quota expression.
- Core Execution v7 adds typed integer literals. The real Go helper contains a
  nested `+ 0`, which survives canonical emission and lowers to `i64.const 0`
  before executing through the unchanged application behavior.
- Provider ingestion projects package-level function declarations across the
  module with package-qualified stable identities and deterministic evidence.
- Actual Pulp loads the same reactor once, processes repeated structured quota
  requests through `pulp_on_call`, returns responses, delivers every granted
  effect, and rejects the identical artifact when the capability is absent.

- Analyze the transitive package closure against a versioned Wasm target
  contract.
- Resolve every requirement as exact, refined, adapted, emulated, embedded
  runtime, native island, or impossible.
- Emit and execute a faithful Wasm component when the plan permits it. Complete
  for the scoped core-module profile; Component Model packaging remains.
- Reject or expose mixed deployment when semantics cannot remain all-Wasm.
- Require no source changes made merely to cater to WebAssembly.

Additional projections, mechanics, ecosystems, and targets follow these proofs
and use the same provider, patch, target-contract, and conformance boundaries.
