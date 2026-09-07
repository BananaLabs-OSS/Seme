# Language bridge roadmap

## Product promise

Seme preserves program meaning independently of source-language syntax. A
supported ecosystem provider lifts native source into canonical Seme semantics;
a projection renders those semantics as idiomatic native source; and a target
realizes them in a native runtime, WebAssembly, Pulp, or another declared
environment.

This is not a promise that every language construct is interchangeable. Every
mapping reports one of the following outcomes:

- exact;
- refined;
- adapted;
- emulated;
- embedded runtime;
- native island;
- remote provider; or
- impossible.

Unsupported meaning must remain opaque when preservation is sufficient, or be
rejected when executable understanding is required. It must never be guessed.

## Adapter topology

Providers connect ecosystems to canonical Seme semantics. Seme does not build a
quadratic matrix of direct language-to-language translators.

```text
Go ---------+
JavaScript -+-> canonical Seme -> Go projection
C# ---------+                  -> JavaScript projection
Blueprints -+                  -> native and portable targets
```

The canonical graph, semantic contracts, provenance, identities, fidelity, and
evidence are durable. Source syntax and target layout are projections.

## Generic execution sequence

The frozen Core Execution v1-v7 proofs remain compatibility gates. New work
must replace whole-function recognition with compositional semantics:

1. ordered blocks, returns, and generic expression trees;
2. local bindings, places, reads, and assignment;
3. structured conditionals, then loops and control transfer;
4. semantic operations and calls resolved through versioned contracts;
5. declared state with initialization, mutability, lifetime, isolation, and
   concurrency semantics;
6. tuples and multiple results, records, Result and Option;
7. arrays, lists, slices, maps, and their explicit mechanics;
8. functions, closures, methods, interfaces, and dispatch;
9. effects, capabilities, errors, asynchronous behavior, and concurrency.

Every expression and statement has a stable identity, an explicit or validated
type, source provenance, and a fidelity result. Mutation uses explicit places;
global reads are not disguised local values; and effects are not disguised
ordinary calls.

## Go provider

The Go provider will use Go's parser and type information to lift individual
typed syntax nodes compositionally. Each node produces canonical meaning,
requirements, fidelity, and located diagnostics. Existing bounded profiles stay
as frozen conformance cases, not as the architecture for new support.

The provider must prove:

- type-checked ingestion of real package closures;
- stable identity across formatting and native renames;
- native Go behavior against the canonical interpreter and target realizations;
- minimal, revision-bound projection back to ordinary Go;
- re-ingestion yielding the intended canonical revision; and
- explicit rejection of unsupported executable subtrees.

## JavaScript provider

JavaScript connects to the same canonical semantics rather than translating
through Go. Its provider and projection must preserve JavaScript-specific
meaning, including Number versus BigInt, coercion and truthiness, null versus
undefined, UTF-16 string observations, objects, arrays, functions, prototypes,
exceptions, promises, and asynchronous effects as support expands.

The first proof is deliberately bounded: canonical functions, booleans, text,
BigInt-backed signed integers, records/objects, arrays, branching, and calls.
Generated JavaScript must be idiomatic, execute under Node, re-lift to equivalent
canonical meaning, and report every adaptation.

## Downstream consumer boundary

Applications are built using Seme. They are not represented as features,
fixtures, or special cases inside the Seme repository.

Each downstream application owns an external acceptance harness that references
its actual source files and declarations. Each run records repository revision,
dirty state, relative source path, file and declaration digests, provider
revision, and evidence. No application function body is copied into Seme.

The acceptance ladder begins with small pure declarations and grows through
text/path behavior, local calls, records, collections, state, capabilities, and
the application boundary. Generic Seme capabilities are added only when a real
source construct exposes a missing semantic building block.

## Correctness gates

For each supported semantic increment:

1. parse and type-check native source;
2. lift and validate the canonical graph;
3. execute native and canonical implementations over boundary and generated
   inputs;
4. execute each claimed target and compare results and effect traces;
5. project to each claimed native language, execute it, and re-lift it;
6. compare canonical meaning independently of formatting;
7. verify deterministic artifacts and revision-safe identities;
8. verify intentionally broken implementations are detected; and
9. verify unsupported constructs fail with located diagnostics.

Passing a bounded profile proves only that profile. Broader language support is
never implied by compilation success alone.

## Documentation and environment record

Every milestone records semantic schemas, fidelity decisions, tests, generated
artifacts, external tool requirements, installs, environment settings, and known
unsupported behavior. Repository-specific consumer evidence remains in the
consumer repository.
