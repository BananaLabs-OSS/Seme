# Core Execution Semantics v27

Core Execution v27 introduces bounded dynamic dispatch without adopting Go
interfaces, JavaScript prototypes, class inheritance, vtables, or a target ABI:

- `InterfaceType` identifies an ordered set of required methods.
- `MethodRequirement` specifies a name, ordered parameter types, and result.
- `SatisfactionWitness` explicitly binds a concrete type's methods to an
  interface's requirements.
- `InterfaceValue` carries an interface type, concrete value, and witness.
- `DynamicMethodCall` selects a requirement from that value and supplies
  ordered arguments.

Satisfaction is evidence in the semantic graph, not an inference deferred to a
target. A target must verify the witness's concrete and interface types, method
count and order, receiver type, parameter and result signatures, and requested
requirement membership before dispatch. Unknown runtime tags and mismatched
witnesses reject rather than selecting a default implementation.

The first bounded proof defines `Adjuster.Adjust(i64) -> i64` and two
implementations: `OffsetAdjuster` adds stored state while `ScaleAdjuster`
multiplies by stored state. A deterministic tag chooses the concrete interface
value. Native Go interfaces and native JavaScript object/class dispatch must
lift to equivalent canonical meaning, project without drift, and agree through
standalone Wasm and pinned Pulp for both implementations, signed values, and
modular-i64 overflow.

This profile does not yet claim interface inheritance, empty interfaces,
runtime type assertions, reflection, method overloading, optional methods,
default methods, variance, pointer-method sets, or unbounded dynamic loading.
