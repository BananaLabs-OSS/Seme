# Generic pure-function Pulp cell

This target contains the checked Core Execution v12 WebAssembly realization of
a generic structured pure function. Its provider ABI is derived from canonical
parameter order and types. Pulp sees only provider identity and opaque bytes.

The checked fixture has request layout `bool, i64, i64` (17 bytes) and one
canonical Boolean response byte. This is conformance evidence for the bounded
v12 ABI, not a universal serialization format.
