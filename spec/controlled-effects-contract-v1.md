# Controlled Effects Contract v1

Controlled Effects v1 has module identity
`00000000000000000000000000013000` and revision identity
`00000000000000000000000000013001`. It imports Package v4, Execution v36,
and Semantic Foundation v1.

The contract represents declared control of three inputs to otherwise
deterministic project behavior:

- `ClockSample` supplies an integer Unix-millisecond value and an explicit
  sequence. `ClockAuthority` names its injection and monotonicity policies.
- `SeededRandomState` supplies an integer state and draw count.
  `SeededRandomAuthority` names the exact algorithm and overflow policy and
  binds the exact project-owned function that realizes its next draw.
- `ExternalBooleanEffect` binds one effect identity, one capability, the
  corresponding Execution effect, its Boolean payload type, and a delivery
  policy.

`ReplayStep` records command sequence, exact clock sample, signed random
before/after/draw values, the unsigned draw ordinal, effect value, the exact
canonical command, and SHA-256 evidence for the response, ordered events, and
state before/after. `ControlledReplay` authenticates the initial seed, exact
canonical initial state and its SHA-256, ordered steps, transcript digest,
duplicate policy, and rejection policy. Digests authenticate derived evidence;
they never substitute for the initial state or command bytes needed to replay.
`ControlledEffectsBounds` declares finite step, clock, seed, draw, effect,
command, initial-state, and total-transcript limits. `ControlledEffectsPlan`
binds these authorities to exact apply and replay functions and a content
revision.

An instance must reject missing, duplicated, inconsistent, malformed, or
over-limit authority. Ambient time and randomness have no meaning under this
contract. Capability denial occurs before observable work. Replay consumes
only recorded samples and seeded state and must reproduce state and ordered
effect traces. A requested effect is not evidence of physical delivery; target
and provider evidence must report that boundary separately.

This contract proves no wall-clock accuracy, timezone or calendar behavior,
cryptographic randomness, timers, concurrency, arbitrary payload effects, or
operating-system integration.
