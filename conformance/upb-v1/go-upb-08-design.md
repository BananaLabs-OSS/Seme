# Go UPB-08 acceptance design

Status: unclaimed. This document is an implementation design, makes no
completion claim, and does not change the UPB scorecard.

UPB-08 should extend the complete cumulative UPB-07 project with one bounded,
typed command/event stream. The new contract describes ordering, correlation,
deduplication, and replay meaning. Byte framing, process I/O, sockets, HTTP,
WebSocket, and Pulp calls remain provider or target realizations; none becomes
Seme Core merely because one realization carries the stream.

## Bounded ordinary project proof

The ordinary Go delta should add one project-owned `transport` package and one
thin adapter package. The semantic entry is a pure transition over explicit
stream state:

```text
Dispatch(stream-state, typed-command)
    -> Result<{stream-state, receipt, ordered-events}, typed-error>
```

The command payload invokes the already authenticated UPB-07 durable planner;
it must not introduce a second copy or a transport-specific reimplementation
of that behavior. The bounded event vocabulary reports either the accepted
planner decision or its typed domain rejection. Transport rejection occurs
before domain dispatch and emits no domain event.

The fixture remains ordinary Go. It imports its own packages normally, passes
offline `go test ./...`, and contains no Seme artifacts, identity constants,
canonical-wire decoder, Wasm ABI, or projector conditionals.

## Fixed stream model and limits

One authenticated Transport v1 selection fixes these finite limits:

- one stream identity, encoded as non-empty UTF-8 text of at most 128 bytes;
- one correlation identity, 16 opaque bytes, compared only for equality;
- command sequence numbers in `[1, 256]`, with the first expected value one;
- at most 256 accepted unique commands in one stream;
- command-kind identity plus a typed canonical payload of at most 3,072 bytes;
- at most four events per newly accepted command;
- event sequence numbers in `[1, 1024]`, allocated globally per stream;
- an event-kind identity plus at most 3,072 combined event-payload bytes in one
  response batch;
- at most 4,096 bytes for one complete encoded command frame and one complete
  encoded response frame;
- no opaque byte value over 4,096 bytes, list over 512 members, or typed-value
  nesting over 32 levels, matching the current Pure Value ABI bounds; and
- a retained receipt and event batch for every accepted command in the bounded
  stream, permitting exact duplicate responses during this session.

The implementation may choose smaller fixture payloads but may not silently
raise, omit, or interpret these authenticated bounds. Integer overflow at any
sequence or size calculation rejects before mutation. The stored `next`
command/event counters may hold the terminal sentinels 257 and 1,025 after
the corresponding final permitted allocation; neither sentinel is a valid
envelope sequence.

Stream state contains the next expected command sequence, next event sequence,
and an ordered retained-command ledger. Each ledger entry owns the exact
correlation identity, command kind, canonical payload digest, receipt, and
ordered emitted events. It is explicit canonical state, not an ambient Go map,
goroutine, process counter, clock, or transport connection property.

## Ordering, correlation, and duplicate policy

For expected command sequence `N`, behavior is exactly:

| Input | Result | State and effects |
|---|---|---|
| `sequence == N`, valid new envelope | dispatch once, retain receipt/events, advance to `N+1` | one atomic transition |
| `sequence < N`, byte-identical authenticated command identity and payload | return the retained receipt/events | no dispatch, state change, domain port call, or newly emitted domain effect |
| `sequence < N`, but any correlation, kind, or payload differs | `transport.sequence_conflict` | none |
| `sequence > N` | `transport.out_of_order` | none; no buffering |
| `sequence == 0` or over the bound | `transport.sequence_invalid` | none |

“Byte-identical” above means equality of the normalized typed envelope: stream,
sequence, 16-byte correlation identity, semantic command kind, canonical
payload bytes, and payload digest. Alternate or noncanonical wire encodings are
rejected rather than normalized into duplicates.

An exact duplicate's returned receipt and event batch are byte-identical cached
response data. They are not newly emitted semantic events or effects. At the
host boundary, returning those bytes may still perform exactly one declared
transport-send operation; that transport trace is reported separately from the
unchanged domain trace.

Correlation identities are caller-supplied opaque values. Every receipt
repeats the exact correlation identity. Every newly emitted event carries the
same correlation identity, the accepted command sequence, a zero-based ordinal
within that command, and its unique global event sequence. Correlation does not
replace ordering: reusing a correlation identity at another command sequence
is rejected as `transport.correlation_reused`. A correlation identity is never
generated, parsed, sorted, truncated, or inferred by Seme.

New-command dispatch is atomic. Either the returned next stream state and its
complete event batch commit together, or neither commits. A domain rejection
is a successfully ordered command outcome: it receives and retains a typed
rejection receipt, consumes the command sequence, and may emit the one declared
rejection event. A malformed, unauthorized, oversized, duplicate-conflicting,
or out-of-order transport input is not a command outcome and consumes nothing.

## Replay definition

The replay artifact is an authenticated initial state plus the ordered list of
accepted unique canonical command envelopes. Receipts and events are derived
observations, not replay inputs. Starting from the same initial application,
durable-planner, and empty stream state, replay must reproduce byte-for-byte:

- every intermediate and final stream state;
- each typed receipt and ordered event batch;
- event and command sequence allocation;
- the accepted-command ledger; and
- the final canonical state and transcript digest.

Exact duplicate attempts, gaps, malformed frames, and other rejected transport
inputs may be retained in a diagnostic process transcript, but they are not
accepted replay commands and cannot affect semantic replay. Replaying a replay,
or replaying its canonical encode/decode result, must produce the same final
bytes. Recovery across process restart, log compaction, acknowledgement,
delivery guarantees, multiple streams, and distributed consensus are outside
UPB-08.

## Neutral Transport v1 contract

The first neutral contract should authenticate meaning rather than a network
API:

- stream, correlation, command-sequence, and event-sequence types;
- typed command and event envelope schemas;
- typed success/rejection receipt schemas and stable error identities;
- exact command-kind and event-kind registries with typed payload ownership;
- a deterministic pure dispatch callable and its state/result types;
- first/next sequence, event ordinal, duplicate, conflict, gap, and correlation
  policies;
- command, event, ledger, payload, and encoded-frame bounds;
- a canonical payload codec and digest algorithm identity;
- an ordered accepted-command replay schema and transcript digest;
- distinct transport-receive and transport-send capabilities for the host
  adapter, both separate from domain and durable-state capabilities;
- one provider requirement describing whole-frame receive and whole-frame send;
  and
- a Project revision binding Transport v1 and its exact provider requirements
  above the complete Project-v10 snapshot.

Whole-frame receive returns either one complete bounded byte string or a typed
framing error. Whole-frame send accepts one complete bounded response or fails
without claiming that any response was delivered. The neutral contract does
not promise packet boundaries, retransmission, connectivity, backpressure,
encryption, authentication, or remote identity.

The selected host codec should be one small deterministic length-prefixed
binary format derived from the authenticated typed envelopes. Its decoding is
strict: exact magic/version, fixed endianness, minimal lengths, valid UTF-8
where the type requires text, known kind, exact field count/order, exact total
length, and no trailing bytes. Application Wire v2 may be reused only where its
actual record shape and authenticated bounds match; it must not be mislabeled
as a general event-stream format. Canonical Seme storage wire is never exposed
as an untrusted application protocol.

## Exact execution routes

A new valid frame follows this order:

```text
authorize transport receive + send
  -> receive complete bounded frame
  -> strict frame decode
  -> validate typed envelope and authenticated digest
  -> classify new / exact duplicate / conflict / out-of-order
  -> for new: pure Dispatch exactly once
  -> atomically commit stream state plus complete event batch
  -> encode the typed correlated response
  -> send one complete response frame
```

An exact duplicate follows decode and classification, then encodes the retained
response without calling the domain, durable port, clock, randomness, or any
other effect. Conflicts and gaps stop after classification. Malformed and
oversized frames stop before envelope classification. Missing capability stops
before receive or send.

The semantic commit precedes host send. Therefore a send failure may leave one
accepted command in authoritative stream state while producing no successful
send observation. Retrying the exact command must return the retained response
without re-execution. The gate must report this honestly; it must not roll back
domain state after a host delivery failure or claim exactly-once network
delivery. A provider that reports successful send must expose independent
evidence that the exact requested response bytes were accepted.

## Placement and target claims

The pure stream classifier/dispatcher/replay program is claimed for native Go,
canonical execution, standalone Wasm, and the pinned Pulp runtime, using the
same canonical program and typed corpus. It receives all state explicitly and
performs no transport or durable I/O.

The framed TransportPort executor is initially claimed only at an
authenticated Go host boundary. Its immutable profile must be derived from the
validated Transport-v1 artifact and must drive codec identity, kinds, bounds,
capability, and receive/send operation ordering. Pinned Pulp may claim the
framed executor only if an actual provider supplies the same whole-frame,
bounded, atomic-send contract and independent saved-byte evidence. HTTP,
WebSocket, `pulp.call_raw`, stdin/stdout, or a socket is not automatically that
provider. Otherwise Pulp must resolve the host adapter as unsupported or a
visible native island while continuing to execute the pure planner exactly.

## Deterministic corpus

Retain the 2,080 UPB-07 durable planner cases as an independent cumulative
baseline. Add a deterministic UPB-08 semantic corpus with at least 4,096
stream observations, including:

- the first, middle, and final permitted unique command;
- success and typed domain-rejection commands;
- zero-, one-, and maximum-event batches;
- exact duplicate before and after other accepted commands;
- conflicting correlation, kind, payload, and digest for an old sequence;
- reused correlation at a new sequence;
- a one-step and maximum gap, followed by the still-valid expected command;
- command/event sequence and arithmetic boundaries;
- empty and maximum-size typed payloads; and
- a fixed generated mix whose seed and generator algorithm are committed.

Each observation separates input classification, returned typed result,
previous/next authoritative state, receipt, ordered events, accepted replay
entry, ordered host trace, and other effects. Two independent native corpus
generations must be byte-identical. Native, canonical, Wasm, and Pulp must
consume identical encoded inputs and match every semantic observation.

A separate finite framing corpus covers every frame boundary and adversary. It
must test the actual streaming reader in deliberately fragmented reads and
coalesced multi-frame reads; a slice-only decoder is insufficient transport
evidence.

## Reuse versus new work

Reuse unchanged:

- the complete UPB-07 ordinary project, 2,080-case planner corpus, Durable-v1,
  Source Presentation-v1, Project-v10, projection, and host DurablePort proof;
- Execution v36 values, explicit state transitions, results, effects, records,
  bytes, strings, slices, maps, and checked control flow where sufficient;
- the canonical evaluator, standalone Wasm cell, pinned Pulp runner, source
  inventory, package/dependency/configuration/resource evidence, deterministic
  bundle publisher, and create-only projector;
- Pure Value ABI and existing Application Wire contracts where their declared
  shapes genuinely apply; and
- the complete claimed UPB01--07 runner as regression evidence.

New, if the bounded proof demonstrates the need:

- Transport v1 contract plus strict validator/emitter;
- the Project revision that binds it above Project v10;
- one strict Go transport selection/adapter;
- pure stream transition and replay packages;
- one bounded streaming host codec and authenticated TransportPort executor;
- source-free loader/report fields for transport authority and placement; and
- provider, evaluator, or target support only for general semantic forms
  exposed by the ordinary fixture.

No fixture package path, command/event name, field name, error number, stream
identity, or corpus vector may be recognized by Core, the Go provider,
canonical evaluator, Wasm lowering, Pulp runner, or generic transport runtime.

## Seven acceptance evidence classes

1. **Native project.** The complete ordinary Go project passes offline before
   lift. Independent corpus builds match byte-for-byte. Native tests cover the
   pure state machine, actual fragmented/coalesced framing, send failure plus
   exact retry, and the unchanged UPB-07 behavior.
2. **Project lift.** One authenticated build owns every stream state, envelope,
   receipt, replay, dispatch, command-kind, and event-kind declaration in its
   real package and source origin. The new Project revision binds exactly one
   Transport-v1 instance to the complete Project-v10 snapshot without
   flattening ownership or introducing a synthetic composition root.
3. **Canonical parity.** Native and canonical execution agree on every new,
   duplicate, conflicting, out-of-order, correlation-reuse, domain-rejection,
   boundary, and generated semantic case: values, previous/next state,
   receipts, ordered events, replay entries, typed errors, and absence of
   undeclared effects all match.
4. **Resolution fidelity.** A source-free loader revalidates the entire prior
   authority chain plus Transport v1, exact callable/type/kind ownership,
   codec/digest identities, bounds, capability, provider operations, ordering,
   placement, and Project binding. It emits a deterministic report and rejects
   any unsupported target rather than upgrading its fidelity.
5. **Target and host-boundary parity.** Standalone Wasm and pinned Pulp execute
   the same pure stream/replay program and match all semantic observations.
   Separately, an authenticated host profile drives the actual bounded
   fragmented-frame executor and exact receive/send traces. Any Pulp framed
   TransportPort claim requires a conforming real provider and independent
   exact-byte evidence; otherwise the report proves its visible unsupported or
   native-island placement.
6. **Project round trip.** Authenticated projection emits ordinary buildable Go
   with native tests, aliases, opaque files, and resource bytes preserved under
   their contracts. Re-import reproduces exact execution meaning and normalized
   Transport authority. Source/inventory-bound revisions change honestly, and
   a second projection/re-lift reaches a byte-identical fixed point.
7. **Atomic rejection.** Every adversary below rejects without a partial
   bundle, report, projection, successful response frame, semantic state
   transition, replay append, domain/durable call, or unrelated effect. When
   failure can be learned only after an accepted semantic commit (host send
   failure), the committed state, exact bytes offered, and any
   provider-reported partial physical write remain visible; retry proves no
   second dispatch.

## Required adversaries

Contract and project authority:

- absent, duplicate, unknown, cyclic, or over-limit stream/kind identities;
- missing or duplicate command/event envelopes, receipts, replay state, or
  dispatcher;
- wrong field types, order, owners, visibility, imports, source origins,
  effects, or callable signatures;
- zero, inconsistent, overflowing, or silently widened limits;
- unknown codec/digest, noncanonical codec policy, or mismatched payload type;
- missing, duplicate, extra, or wrong transport capability and receive/send
  operations;
- reversed or duplicate operation order;
- Transport-v1 or Project artifact tamper; independently valid artifacts mixed
  across builds; and an extra, missing, renamed, or unauthenticated bundle
  member.

Semantic stream and replay:

- sequence zero, maximum plus one, integer overflow, gaps, old unknown sequence,
  and attempts to buffer or later auto-apply a gap;
- exact duplicate that dispatches, emits, appends replay, changes state, calls
  DurablePort, or allocates new event sequences;
- old-sequence changes to stream, correlation, kind, typed payload, canonical
  bytes, or digest;
- correlation reuse at a new sequence and empty/short/long correlation bytes;
- wrong event correlation, cause sequence, ordinal, global sequence, kind,
  count, or order;
- partial batch/state/replay commit on dispatcher rejection or arithmetic
  overflow;
- replay containing duplicates, gaps, reordered commands, derived receipts or
  events as inputs, wrong initial state/digest, or a forged final digest;
- replay whose first and second executions differ; and
- ambient clock, randomness, goroutine scheduling, map iteration, network,
  filesystem, environment, or global mutable state in the pure package.

Framing and host execution:

- empty, truncated at every field, bad magic/version, unknown flags/kind,
  nonminimal length, integer overflow, impossible nested length, invalid UTF-8,
  wrong correlation length, digest mismatch, noncanonical payload, duplicate
  field, wrong order, trailing byte, and multiple values in one frame;
- payload, event, response, and total-frame sizes exactly at and one byte over
  each bound;
- one frame split at every byte boundary, two frames coalesced in one read,
  short reads, read error, EOF mid-frame, and extra bytes retained for the next
  frame;
- unauthorized execution (including capability granted only after receive),
  wrong authenticated profile, and profile mutation after construction;
- zero sends on pre-dispatch failure, exactly one complete send attempt after
  accepted dispatch, short/partial provider send reported as success, and a
  provider claiming success without retaining the exact requested bytes;
- send failure followed by exact retry, conflicting retry, or out-of-order next
  command; only the exact retry may recover the retained response without new
  semantic work;
- malformed Wasm ABI, denied target capability, trap, or partial target output;
  and
- a Pulp plan that silently equates HTTP, WebSocket, raw cell calls, files, or
  process stdio with the authenticated TransportPort contract.

Bundle and projection adversaries inherited from prior cells (path traversal,
symlinked source/destination parents, digest drift, orphan blobs, existing
output, and create-only failure cleanup) remain required cumulative regression
evidence and must not be redefined as stream semantics.

## Planned commands and claim rule

Development evidence should be split so failures remain attributable:

```text
scripts/check-go-upb-08-fixture.sh         # native project and deterministic corpora
scripts/check-go-upb-08-runtime.sh         # pure canonical/Wasm/Pulp stream parity
scripts/check-go-upb-08-port-runtime.sh    # authenticated framed host boundary
scripts/check-go-upb-08.sh                 # mapped cumulative seven-class gate
```

Only the final cumulative gate may map or claim Go UPB-08. It must execute the
mapped UPB-07 gate as its cumulative predecessor, all three UPB-08 evidence
partitions, deterministic build and source-free report,
projection/native/re-lift fixed point, and every required adversary. Only after
that gate is green and the nonrecursive complete claimed runner executes every
mapped UPB01--08 gate may the mapping, roadmap, scorecard, and claimed runner
move from 7/36 overall and 7/12 Go to 8/36 overall and 8/12 Go.
