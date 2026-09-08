# Core Execution Semantics v18

Core Execution v18 introduces explicit, typed mutation without encoding source
syntax into canonical Seme. `MutablePlace` owns a stable lexical identity, type,
and initializer. `DeclarePlace`, `PlaceRead`, and `AssignPlace` preserve ordered
state transitions. `While` refers to a Boolean condition and a nested block;
the condition is re-evaluated before every iteration.

Ordinary Go condition-only `for` loops and JavaScript `while` loops are native
source views of the same bounded semantics. Go `:=` declarations are lifted as
mutable only when their object is assigned later; JavaScript `let` is explicit.
Both providers converge on byte-identical canonical programs, and JavaScript
projection emits normal `let`, assignment, and `while` syntax that re-lifts
without drift.

The first pure Wasm realization supports string and Boolean places in a
single-function exact-text ABI v2 program. Certification rejects reads before
declaration, duplicate placement, out-of-scope assignment, malformed blocks,
cycles, and traversal overflow before emission. The target emits real Wasm
locals and structured loops. To keep hostile or accidentally nonterminating
programs bounded, a loop traps after 10,000 entered iterations; this is an
explicit realization limit, not a claim that canonical `While` terminates.

Integer places, Go three-clause and range loops, `break`, `continue`, branching
inside loop bodies, mixed immutable/mutable locals, mutable callees, and
concurrent or shared places are not claimed by this profile.
