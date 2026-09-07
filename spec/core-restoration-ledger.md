# Core semantic restoration ledger

This ledger tracks the additive restoration of generic Core Execution
capabilities from v8 onward above the clean Core v7 baseline. It is a planning and acceptance
record, not evidence that a capability exists. A milestone becomes `complete`
only after all evidence named below is checked in and its cumulative gates pass.

The restoration deliberately preserves the semantic families previously
explored while replacing whole-function source-shape selection with
compositional lifting and lowering. Source providers recognize typed language
constructs, emit canonical nodes independently, and compose those nodes through
ordinary references. Targets select behavior from the validated canonical
graph, never from project names, function names, fixture paths, or source ASTs.

## Status vocabulary

- `pending`: no restored implementation is claimed.
- `in progress`: implementation or evidence exists, but the complete gate has
  not yet passed.
- `complete`: the additive module, providers, targets, runtime proof,
  compatibility proof, and documentation all pass.
- `blocked`: a recorded external dependency prevents completion.

## Shared acceptance requirements

Every milestone must provide:

1. An additive canonical module under `modules/execution/vN/`, including its
   readable declaration, compiled artifact, pinned hashes, and module notes.
2. A normative semantic specification and freeze audit under `spec/`.
3. Schema-generation and canonical-validation tests proving stable identities,
   field shapes, reference constraints, and preservation of all earlier Core
   revisions.
4. Provider tests based on type-resolved ordinary source, including positive,
   equivalent-spelling, and explicit unsupported cases.
5. Canonical inspection evidence demonstrating that meaning is recoverable
   without consulting source syntax.
6. Interpreter or target lowering tests derived solely from the validated
   graph, with malformed-graph rejection, bounded traversal, and cycle checks
   where graphs may recurse.
7. Native-oracle differential vectors covering normal, boundary, negative,
   empty, and overflow behavior as applicable.
8. Deterministic artifact generation, including two independent lowerings with
   byte-identical results.
9. Execution through standalone WebAssembly and actual Pulp, including invalid
   ABI input rejection and capability denial whenever effects are involved.
10. Revision-bound semantic edit evidence and native test preservation where
    the milestone introduces editable declarations or references.
11. The complete frozen product gate and cumulative external corpus gate, with
    one isolated, ordinary semantic-family project added when new source
    behavior is introduced.
12. An environment-change statement recording installs or configuration
    changes, or explicitly recording that none were required.

Tests and examples must use generic semantic terminology. Consumer
applications remain external acceptance subjects and must never be copied into
the Core, provider, target, or conformance fixtures.

## Restoration milestones

| Core | Status | Semantic family | Compositional replacement and milestone-specific evidence |
|---|---|---|---|
| v8 | in progress | Compositional function bodies | `Block` and `Return` are implemented as provider-neutral ordered statement and returned-expression references. Additive declarations, typed lifting, deterministic graph generation, canonical validation, bounded checking, and frozen v1-v7 construction pass. The v12 certificate and target now execute this structure without selecting source names. Completion still requires dedicated canonical inspection evidence, revision-bound edit evidence, and the complete cumulative product and external-corpus gates required above. |
| v9 | in progress | Typed integer multiplication | `IntegerMultiply` is implemented with typed recursive lifting, canonical validation, nested evaluation, signed modular-i64 native parity, overflow vectors, deterministic graph generation, recursive Wasm instruction lowering, and frozen v2-v8 construction. The v12 certified backend accepts the node generically. Completion still requires direct standalone-Wasm and actual-Pulp multiplication vectors through that backend, canonical inspection and revision-bound edit evidence, invalid-ABI coverage for this result profile, and the cumulative product and external-corpus gates. |
| v10 | in progress | Typed integer subtraction | `IntegerSubtract` preserves ordered operands and is implemented across typed lifting, canonical validation, nested evaluation, native parity, underflow vectors, deterministic graph generation, and recursive lowering. The v12 gate now executes subtraction in standalone Wasm and through the pinned Pulp path. Completion still requires dedicated canonical inspection and revision-bound edit evidence, fuller boundary vectors through the runtime ABI, and the cumulative product and external-corpus gates. |
| v11 | in progress | Boolean literals and conjunction | `BooleanLiteral` and short-circuit `BooleanAnd` are implemented with typed lifting, canonical-shape rejection, nested semantic evaluation, native parity, and control-flow Wasm lowering. The v12 runtime gate proves canonical Boolean request/result bytes, rejects malformed Boolean input, exercises both outcomes, and runs through standalone Wasm and pinned Pulp. Completion still requires dedicated canonical inspection and revision-bound edit evidence, an end-to-end runtime proof that an invalid right operand is skipped only on the false path, and the cumulative product and external-corpus gates. |
| v12 | in progress | Generic pure-function cell ABI | Canonical parameter/result types now derive an explicit versioned ABI with ordered field layouts, canonical and artifact provenance, deterministic bytes, bounded allocation/free behavior, standalone execution, and execution through an exact pinned capability-free Pulp runner. An opaque certificate rejects malformed membership, ordering, statement, typing, purity, cycle, and traversal-budget cases before emission; request decoding fails closed. Completion still requires independent canonical inspection and revision-bound edit evidence, a second independently authored semantic source exercising the reusable provider contract, and the complete cumulative product and external-corpus gates. |
| v13 | in progress | Total-return branching and closed text expressions | `If` supplies two explicit nested `Block` branches, `BooleanOr` preserves left-first short-circuiting, and `StringConcat`/`StringEqual` compose closed, validated UTF-8 literal expressions using exact byte semantics. Typed lifting, recursive validation, deterministic artifacts, native differential vectors, standalone Wasm, actual Pulp execution, bounded constructed text, and frozen v2-v12 generation are implemented. The executable profile deliberately requires both branches to return and does not expose variable-width text parameters or results. Completion still requires dedicated canonical inspection and revision-bound edit evidence, malformed-graph tests spanning every new node and nested control-flow failure, broader UTF-8 and allocation boundaries, and the cumulative product and external-corpus gates. |
| v14 | pending | Effect invocation and sequencing | Add `EffectInvoke` with a stable effect identity, ordered argument expressions, and continuation/result reference. Resolve capabilities during target planning; prove granted delivery, ordered traces, and denial traps without importing target capability schemas into the execution module. |
| v15 | pending | Locals and structured choice | Add stable local declarations, `LocalRead`, scoped `Let`, strict integer comparison, and non-total conditional forms. Use lexical references rather than source names; prove scope rejection, fallthrough analysis, and positive and negative branches. |
| v16 | pending | Fixed arrays and indexed reads | Add element-typed, fixed-length array values and index reads. Derive ABI layout from canonical type and prove signed values, boundary indexes, overflow-preserving elements, and out-of-range rejection. |
| v17 | pending | Bindings and collection fold | Represent range bindings and fold state explicitly, then compose fold body expressions from ordinary canonical nodes. Prove empty/non-empty arrays, binding scope, deterministic iteration order, and modular accumulation. |
| v18 | pending | Runtime-sized slices | Add `SliceType` and length-prefixed runtime values independently of fixed arrays. Prove empty and variable lengths, allocation bounds, malformed lengths, dynamic iteration, and Pulp request decoding. |
| v19 | pending | Collection length and computed indexing | Add `CollectionLength` and `DynamicIndexRead` applicable through type-directed collection contracts. Prove computed last-element access, empty fallback, runtime bounds checks, and composition with locals and conditionals. |
| v20 | pending | Append and collection update | Add immutable semantic append/update operations with an explicit resulting collection. Prove capacity-independent meaning, dynamically sized responses, ordered updates, alias-safe lowering, and bounded allocation failure. |
| v21 | pending | Methods and explicit state transitions | Add receiver types, receiver-field reads, method calls, and explicit updated-state-plus-result transitions. Prove value and pointer receiver behavior, multiple sequential calls, state threading through the ABI, and identity-safe rename across call sites. |
| v22 | pending | Interfaces and bounded dynamic dispatch | Add interface types, explicit satisfaction witnesses, interface values, and dynamic calls. Prove at least two implementations, deterministic dispatch tags, signature validation, unknown-tag rejection, and calls selected from canonical witnesses rather than source naming. |
| v23 | pending | Immutable lexical closures | Add function types, capture declarations, capture reads, closure construction, and indirect calls. Model the environment explicitly; prove returned closures, positive and negative captured values, invocation validation, and environment lifetime through Pulp. |
| v24 | pending | Mutable closure environments | Add capture updates, sequencing, and explicit committed environment state. Prove multiple invocations observe prior updates, independent closure instances do not alias, negative-state vectors, and deterministic state/result encoding. |
| v25 | pending | Runtime-keyed maps | Add allocation, runtime-key lookup, zero-value missing lookup, and update for bounded `map[int64]int64`. Prove repeated keys, negative keys, deterministic logical results independent of physical ordering, allocation limits, and a composed tally/fold workflow. |

## Architectural completion condition

Restoring the table is necessary but not sufficient. The restoration is
architecturally complete only when representative programs combine several
families without adding a new whole-function recognizer or target profile. At a
minimum, one cumulative proof must compose:

```text
parameters -> locals -> conditional -> collection traversal
           -> call or dynamic call -> state update -> returned value
```

and a second must compose runtime strings with collection or map behavior.
Adding a schema while retaining syntax-shaped execution shortcuts leaves that
milestone `in progress`.

## Current baseline

Core v7 remains the compatibility anchor. It already supplies typed functions,
records, explicit Results, direct calls, effects used by the bounded
application profile, conditionals, strings and bytes as canonical value types,
parameter reads, integer literals, integer addition, normalized integer
less-than-or-equal comparison, recursive expression emission, and recursive
Wasm lowering. Restoration must be additive and must keep its checked artifacts
byte-identical.
