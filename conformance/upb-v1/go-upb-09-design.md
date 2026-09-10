# Go UPB-09 semantic design

Status: accepted implementation design; unclaimed.

Go UPB-09 extends the cumulative Project-v11 fixture with one project-neutral
Runtime Inputs v1 plan and one Project-v12 binding. It does not revise Core or
Execution and does not add Go, operating-system, or product concepts to Seme.

## Bounded semantic profile

The fixture supplies a typed `ClockSample` containing a millisecond value and
sequence. A new accepted command requires the next sequence and a
nondecreasing value in the closed range `0..4102444800000`. Clock values are
explicit replayable inputs. Calls to ambient Go time, WASI clocks, sleep,
calendar, timezone, or scheduler services are outside this claim.

The random source is a pure versioned state transition. Its seed is an
explicit nonzero signed i64 bit pattern, its draw count is bounded by the
transport's 256 accepted-command limit, and one draw computes
`state = state * 48271 + 1` with Execution's signed-i64 modular arithmetic.
The resulting full i64 bit pattern is the draw. This does not claim host
entropy, cryptographic randomness, ranges, distributions, or another PRNG.

Each newly accepted ordered command consumes exactly one valid clock sample
and one random draw, invokes the existing domain planner once, records the
result atomically, and requests exactly one ordered
`observability.log(bool)` effect. The Boolean records domain acceptance.
Domain rejection consumes the accepted command's clock/random inputs and
requests `false`. Exact transport duplicates return cached authority and
consume no clock, random draw, or effect. Transport or clock rejection changes
nothing and requests no effect.

All reachable capabilities are preauthorized before evaluation or host calls.
Missing, extra, ambiguous, or substituted authority rejects before observable
work. Live host execution obtains one declared clock observation and may
deliver one declared effect. A delivery result is reported separately from the
atomic semantic transition; no exactly-once external-world guarantee is
claimed.

## Replay

Replay contains the initial state/seed and every accepted unique command with
its exact clock sample, random-before/random-after values, draw ordinal,
effect request, response/event authority, and digests. Replay verifies those
records and reproduces every intermediate state, final state, response, event,
random draw, and ordered logical effect trace. It performs zero live clock,
entropy, transport, storage, or external-effect operations.

## Runtime placement

Native Go, canonical evaluation, standalone Wasm, and pinned Pulp execute the
same explicit-input pure planner and replay. The pinned Pulp runtime's ambient
WASI clock and `entropy.read` are rejected as substitutions: neither realizes
this declared clock/seeded-random profile. Until separately pinned providers
exist, live clock sampling and effect delivery form an authenticated Go host
boundary. Pulp's existing synchronous opaque call carrier proves the pure
planner only.

## Evidence requirement

The cumulative gate must prove all seven UPB evidence classes and at least
4,096 deterministic observations, including seed extrema, modular overflow,
first/final draws, minimum/maximum/equal/increasing clock values, domain
rejection, exact duplicates, gaps, correlation conflicts, replay, and terminal
sentinels. Native, canonical, Wasm, and Pulp state/results/traces must agree.

The gate must also reject invalid seeds; invalid, stale, gapped, or decreasing
clock samples; inconsistent random state/ledger entries; replay omission,
reordering, or tampering; wrong algorithm/version; forged effect identity or
payload; denied capabilities; ambient time/entropy substitutions; mixed or
tampered artifacts; unsafe projection; and existing destinations without
partial graph, artifact, source, state, or effect commits.

The scorecard remains unchanged until the complete cumulative authority gate
passes from a clean commit.
