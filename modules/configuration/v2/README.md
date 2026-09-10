# Configuration Contract v2

Configuration v2 (`...4000@...4005`, parent `...4001`) adds an immutable,
typed argument-binding plan for initialization. Runtime values remain outside
the artifact. A `RuntimeInput` names and types the startup ABI input and may
authorize an ambient provider with a Foundation Capability without storing its
value.

Argument sources are a closed union: resolved configuration field, runtime
input, predecessor success payload, static canonical value, or exact record
construction. Record construction is complete and ordered; partial records do
not imply hidden defaults. Validators must prove exact callable parameters,
types and dependency edges, and reject cyclic source graphs.
