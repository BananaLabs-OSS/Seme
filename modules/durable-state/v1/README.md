# Durable State Contract v1

Durable State v1 (`...8000@...8001`) describes versioned state families,
typed versions, pure validators and migrations, and capability-backed storage
port metadata. Its authenticated operation authority fixes the Load-then-CAS
order, operation and capability identities, canonical codec identity, and the
`opaque-thread-only` comparison-token policy. Typed neutral boundary schemas
cover logical keys, bounded canonical payloads and digests, opaque tokens,
missing/found/error Load outcomes (including the mandatory opaque absence token
for a missing-create CAS), saved/conflict/error compare-exchange
outcomes, and port errors. A port also binds its domain-error type.

Load and atomic compare-exchange are declared Foundation effects; filesystem,
database, retry, transaction, and consistency mechanics remain provider
concerns. The token is deliberately only a value to receive from Load and
thread unchanged into compare-exchange. The contract grants no operation for
inspecting, ordering, deriving, or synthesizing it.

This metadata contract does not certify runtime token opacity, load/CAS trace
ordering, retry behavior, or atomic storage realization. Those properties need
separate runtime evidence from a capability-authorized DurablePort. In
particular, typed metadata does not prove that a host committed the offered
bytes, preserved an opaque token, or emitted the declared trace.
