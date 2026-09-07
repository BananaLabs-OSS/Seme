# Generic pure-function Pulp cell

This target contains the checked Core Execution v12 WebAssembly realization of
a generic structured pure function. Its provider ABI is derived from canonical
parameter order and types. Pulp sees only provider identity and opaque bytes.

The checked fixture has request layout `bool, i64, i64` (17 bytes) and one
canonical Boolean response byte. This is conformance evidence for the bounded
v12 ABI, not a universal serialization format.

The ABI manifest binds its version, provider, target, fidelity, canonical
module/revision/program/function identities, canonical program digest, emitted
artifact digest, and every field's index, offset, size, type, and encoding.

The cell exports a bounded LIFO `pulp_free` paired with `pulp_alloc`. Pulp frees
name, request, out-parameter, and response buffers after each serialized call;
the conformance gate proves 20,000 direct calls return the arena to its initial
state and checks zero, maximum, exhaustion, and reclamation behavior.
Near-maximum forged pointers, pointers above the arena, zero-sized frees, and
out-of-order frees are proven unable to change allocator state before pointer
arithmetic occurs.
