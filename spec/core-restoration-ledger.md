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
| v14 | in progress | Variable-width exact text ABI | The unchanged v13 canonical vocabulary now realizes string parameters and results through `seme.pure-abi/v2`: ordered descriptors, exact UTF-8 scalar bytes, bounded allocation, runtime concatenation/equality, malformed-input rejection, independent JavaScript oracle vectors, standalone Wasm, and pinned Pulp execution pass while scalar ABI v1 remains byte-identical. Completion still requires dedicated canonical inspection and revision-bound edit evidence, wider mixed-signature and malformed-graph coverage, and the cumulative product and external-corpus gates. |
| v15 | in progress | Immutable lexical locals | `LocalBinding`, ordered `BindLocal`, and `LocalRead` explicitly preserve typed lexical identity and single-evaluation meaning. Go `:=` and JavaScript `const` lift compositionally to byte-identical canonical programs; JavaScript projection re-lifts identically; native, standalone Wasm, and Pulp vectors pass for integer and exact-text programs. Certification rejects self/forward reads, duplicate placement, invalid order, type disagreement, and cycles. Completion still requires revision-bound semantic edits, deeper nested-scope vectors, and the cumulative product and external-corpus gates. |
| v16 | in progress | Closed multi-function calls | The existing provider-neutral `FunctionCall` now composes ordinary typed functions into one closed executable program. Go and JavaScript providers produce byte-identical membership, entry, and call graphs; JavaScript projection preserves native module-local calls; the pure target rejects non-members, recursion, excessive expansion, and malformed calls before deterministic target-side inlining and Wasm/Pulp execution. Completion still requires revision-bound call-site edits, branching callee bodies, and the cumulative product and external-corpus gates. |
| v17 | in progress | Compositional immutable records | The `RecordType`, `RecordField`, `RecordConstruct`, and `FieldRead` vocabulary now participates in the generic provider pipeline. Go structs and JavaScript structural objects lift to byte-identical canonical field identities and ordered values; projection remains native; internal record selection executes through bounded certified scalar replacement in Wasm and Pulp. Completion still requires record-valued ABI boundaries, nested records, revision-bound edits, and cumulative corpus gates. |
| v18 | in progress | Mutable places and structured loops | `MutablePlace`, `DeclarePlace`, `PlaceRead`, `AssignPlace`, and `While` preserve typed lexical identity, mutation order, and condition re-evaluation. Go condition-only `for` and JavaScript `while` lift to byte-identical canonical programs and project without drift; string/Boolean places execute as actual Wasm locals and structured loops standalone and through Pulp. Certification rejects lexical misuse, and the target explicitly traps after 10,000 entered iterations. v19 extends the same realization to i64 places. Completion still requires mixed immutable/mutable scopes, branches and calls inside loops, `break`/`continue`, revision-bound edits, and cumulative corpus gates. |
| v19 | in progress | One-sided structured choice and integer mutation | `When` adds a Boolean-guarded fallthrough block without changing total-return `If`. Ordinary Go and JavaScript one-sided conditionals lift to byte-identical canonical programs and project without drift. Signed-i64 places execute both outcomes as real Wasm locals and structured void branches standalone and through Pulp. Completion still requires fallthrough `else`, nested choice/loop composition, arithmetic over place reads, revision-bound edits, and cumulative corpus gates. |
| v20 | in progress | Effect invocation and sequencing | `EffectInvoke` references Foundation effect identities and ordered arguments while block order supplies sequencing. Bounded Go logging and JavaScript console adapters converge on the same observation effect and project without drift. The certified Wasm plan reports adapted fidelity and its required capability; Pulp proves ordered granted delivery and denial trapping with no emitted event. Completion still requires effect results, structured arguments, composition inside control/calls, explicit target-contract resolution entities, revision-bound edits, and cumulative corpus gates. |
| v21 | in progress | Fixed arrays and indexed reads | `FixedArrayType`, `FixedArrayConstruct`, and `IndexRead` now represent element-typed, fixed-length values without adopting Go or JavaScript collection mechanics. Bounded i64 literals and fixed-array parameters lift from both languages to byte-identical canonical programs and project without drift. The packed Wasm ABI is derived from canonical element type/length; exact-size requests and valid boundary indexes agree natively and through Pulp, malformed sizes reject, and negative/upper-bound target reads trap. Completion still requires array-valued results/locals, other element types, mutation, nested arrays, revision-bound edits, and broader cumulative corpus gates. |
| v22 | in progress | Bindings and collection fold | `IterationBinding`, `IterationBindingRead`, and `Fold` represent typed reduction without importing Go loop or JavaScript callback mechanics. Idiomatic Go `range` addition and JavaScript `reduce` lift to byte-identical canonical programs and project without drift. Empty/non-empty fixed arrays, signed values, and modular overflow agree natively and through Wasm/Pulp; binding aliases and unsupported bodies reject. Completion still requires general compositional bodies, behaviorally noncommutative order evidence, index bindings, effects/control inside folds, revision-bound edits, and dynamic collections. |
| v23 | in progress | Runtime-sized slices | `SliceType` represents an element-typed runtime-sized collection without source capacity, object identity, or target storage mechanics. Go `[]int64` and JavaScript `bigint[]` lift to byte-identical canonical folds and project without drift. A bounded descriptor/payload Pure ABI v2 realization dynamically traverses empty, variable, maximum-sized, signed, and overflowing inputs through standalone Wasm and Pulp while malformed layouts, counts, and request sizes reject. Completion still requires slice construction/results, mutation, sub-slicing, non-i64 elements, arbitrary fold bodies, revision-bound edits, and broader cumulative composition gates. |
| v24 | in progress | Collection length and computed indexing | `CollectionLength` and `DynamicIndexRead` query ordered collections without adopting source syntax or target layout. Go `len`/index and JavaScript `.length`/index lift to byte-identical empty-safe last-element programs and project without drift. Pure ABI v2 Wasm dynamically reads validated slice descriptors; empty fallback, valid last access, and negative/upper-bound traps agree natively and through Pulp. Completion still requires wider collection types, index composition with locals/calls, revision-bound edits, slice results/construction, and broader cumulative composition gates. |
| v25 | in progress | Append and collection update | `CollectionAppend` and `CollectionUpdate` produce explicit immutable collection values without capacity or backing-storage semantics. Native Go copying idioms and JavaScript `.with`/`.concat` lift to byte-identical canonical programs and project without drift. Dynamically sized Pure ABI v2 results execute through standalone Wasm and Pulp with fresh storage, preserved inputs, runtime bounds traps, and a 512-element allocation bound. Completion still requires non-i64 and nested collections, revision-bound edits, broader collection composition, and cumulative corpus gates. |
| v26 | in progress | Methods and explicit state transitions | `ReceiverBinding`, `ReceiverRead`, `Method`, `MethodCall`, and explicit transition values preserve receiver type, call arguments, updated state, and returned result without hidden mutation. Native Go value receivers and JavaScript classes project and re-lift to byte-identical canonical bytes; a one-field i64 state executes through a fixed 16-byte Wasm/Pulp ABI with receiver preservation, modular overflow, and malformed-size rejection. Completion still requires pointer-receiver adaptation, sequential transition threading, identity-safe edits, richer states/results, and cumulative corpus gates. |
| v27 | pending | Interfaces and bounded dynamic dispatch | Add interface types, explicit satisfaction witnesses, interface values, and dynamic calls. Prove at least two implementations, deterministic dispatch tags, signature validation, unknown-tag rejection, and calls selected from canonical witnesses rather than source naming. |
| v28 | pending | Immutable lexical closures | Add function types, capture declarations, capture reads, closure construction, and indirect calls. Model the environment explicitly; prove returned closures, positive and negative captured values, invocation validation, and environment lifetime through Pulp. |
| v29 | pending | Mutable closure environments | Add capture updates, sequencing, and explicit committed environment state. Prove multiple invocations observe prior updates, independent closure instances do not alias, negative-state vectors, and deterministic state/result encoding. |
| v30 | pending | Runtime-keyed maps | Add allocation, runtime-key lookup, zero-value missing lookup, and update for bounded `map[int64]int64`. Prove repeated keys, negative keys, deterministic logical results independent of physical ordering, allocation limits, and a composed tally/fold workflow. |

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
