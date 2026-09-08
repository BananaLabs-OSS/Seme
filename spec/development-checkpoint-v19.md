# Seme Development Checkpoint — Core v19

Date: 2026-09-08

This checkpoint records the first continuous cross-language execution stack
through compositional local state. It is a stable resumption point, not a claim
of general-purpose language coverage.

## Product boundary

Seme is a canonical semantic language and compiler system. Developers author
ordinary native source; ecosystem providers lift understood meaning into Seme,
and projectors render canonical programs through native language views. The
canonical program—not a particular source spelling—is compiled.

Consumer applications remain external. No application, editor, game, engine,
or product-specific behavior belongs in Seme's semantic modules, providers,
targets, fixtures, or repository history.

## Implemented semantic foundation

The current compositional path includes:

- typed functions, parameters, results, and closed function calls;
- exact signed modular-i64 and canonical Boolean values;
- exact Unicode-scalar UTF-8 strings, concatenation, and equality;
- total-return branching and nested blocks;
- immutable lexical bindings and reads;
- immutable record construction and field selection;
- typed mutable places, ordered declarations, reads, and assignments;
- structured bounded `While` realization;
- one-sided fallthrough `When` choice;
- canonical program identity, deterministic serialization, and validation;
- certified lowering to standalone Wasm and Pulp cells;
- incremental Go sessions and a deterministic JSON-lines language service.

## Cross-language evidence

The bounded Go and JavaScript providers independently lift native constructs
into byte-identical canonical programs. The JavaScript projector emits ordinary
native JavaScript that re-lifts without semantic drift. The v19 proof covers a
signed-i64 mutable value conditionally replaced through a one-sided branch.

The same behavior passes:

1. the original Go tests;
2. native projected JavaScript execution using `bigint`;
3. canonical graph validation and compilation;
4. standalone Wasm execution for both branch outcomes;
5. execution through a pinned Pulp runtime;
6. canonical Go/JavaScript equality;
7. projection and re-lift equality.

## Certification and safety

The pure target rejects malformed program membership, type disagreement,
invalid lexical order, reads before declaration, duplicate declarations,
out-of-scope mutation, branch-local scope leakage, graph cycles, excessive
graph traversal, malformed ABI input, and non-total functions where a result is
required. Runtime loop realization traps after 10,000 entered iterations rather
than permitting unbounded execution.

## Honest limits

This checkpoint does not claim arbitrary Go or JavaScript. Important remaining
work includes:

- mixed immutable and mutable values across every control shape;
- arithmetic expressions rooted in mutable reads;
- nested mutation across calls and multiple functions;
- fallthrough `else`, `else if`, `break`, and `continue`;
- arrays, slices, dynamic indexing, iteration bindings, and maps;
- methods, interfaces, closures, and broader package ingestion;
- effect invocation and capability-aware sequencing;
- general memory-model, concurrency, and foreign-runtime adapters;
- complete source-preserving projections for unsupported native regions.

Unsupported behavior must continue to reject or remain explicitly opaque. It
must never be guessed or silently weakened.

## Reproducible acceptance

The primary checkpoint gate is:

```sh
./scripts/check-execution-v19.sh
```

The cumulative verification performed at this checkpoint also includes the
complete Go reference suite, Core v18, the incremental Go session gate, and the
language-service JSON-lines gate. Frozen module artifacts include SHA-256
manifests and reproduce byte-for-byte from their generators.

## Repository state

The implementation checkpoint is commit `153b0c0` (`feat: add integer mutation
and structured choice`). At that commit:

- the working tree is clean;
- `git fsck --full --strict` passes;
- generated Core v19 artifacts reproduce exactly;
- current files, filenames, and every reachable commit contain no
  consumer-specific application material;
- no rejected patch, merge-conflict, or synchronization-temporary file remains.

## Environment state

Core v19 used the existing workstation Go 1.26 toolchain, Node 22,
repository-local pinned Acorn dependency, frozen Seme bootstrap, and pinned
Pulp source. No software, dependency, global setting, SSH setting, or
persistent environment configuration was installed or changed.

## Recommended resumption

Resume with effect invocation and capability-aware sequencing as Core v20.
Keep effects in versioned semantic modules, resolve capabilities during target
planning, and prove ordered granted delivery plus denial without importing host
or runtime-specific capability concepts into the language.
