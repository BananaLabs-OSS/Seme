# Controlled Effects Contract v1

This immutable, project-neutral contract describes an explicitly injected
clock sample, a seeded deterministic random state, one capability-authorized
Boolean external effect, and the replay data and bounds that make their use
auditable.

It does not define an ambient clock, entropy source, timer, scheduler, operating
system API, delivery mechanism, or general effect system. A selected instance
must bind exact algorithms, policies, callables, capabilities, and finite
bounds. Unsupported realizations remain visible through Target fidelity.
