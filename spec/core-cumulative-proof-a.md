# Core cumulative proof A: stateful collection flow

This proof combines existing Core v2-v30 semantics without adding vocabulary.
Runtime slice traversal produces a folded sum, a function call binds that value
to an immutable local, a conditional selects a method call, and the method
returns an explicit updated state and result. The input record remains
unchanged.

The fixture is intentionally consumer-neutral. Its Go and JavaScript forms
must lift to identical canonical meaning and execute identically through the
bounded Wasm/Pulp realization.
