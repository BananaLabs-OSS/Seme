# Core Execution Semantics v15

Core Execution v15 adds immutable lexical local bindings to the compositional
statement and expression model. It introduces three provider-neutral schemas:

- `LocalBinding(name, type, initializer)` declares one typed lexical identity;
- `BindLocal(binding)` places that binding at an ordered point in a block; and
- `LocalRead(binding)` reads the value associated with that identity.

A binding enters scope only after its `BindLocal` statement. Its initializer
is evaluated exactly once before later statements. A nested block may read
bindings visible at its entry; bindings declared inside a nested block do not
escape it. Duplicate placement, forward reads, initializer self-reference,
type disagreement, and cycles reject.

The v15 executable profile is deliberately immutable and pure. A target may
inline a binding only when doing so preserves its single-evaluation meaning;
future effects make that restriction observable. Mutable assignment requires
explicit places and state sequencing and is not claimed by this milestone.

Go `name := expression` and JavaScript `const name = expression` are exact
source views for this profile when the binding is not reassigned. Projection
uses those native forms rather than exposing canonical schema syntax.
