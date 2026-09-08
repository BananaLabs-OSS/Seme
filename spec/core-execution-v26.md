# Core Execution Semantics v26

Core Execution v26 introduces methods without adopting a particular source
language's object model:

- `ReceiverBinding` gives a method receiver an explicit typed identity.
- `ReceiverRead` reads that bound value.
- `Method` declares receiver, parameters, result type, and body.
- `MethodCall` supplies an explicit receiver and ordered arguments.

State-changing behavior is represented as data rather than hidden mutation:

- `StateTransitionType` identifies the state and returned-result types.
- `StateTransition` contains the complete updated state and result.
- `TransitionState` and `TransitionResult` select either value.

A source adapter may map a value receiver, immutable object method, state
monad, tuple, or another native convention onto these constructs only when it
can preserve the same observable meaning. Pointer identity, aliasing, implicit
mutation, inheritance, prototype lookup, virtual dispatch, exceptions, and
concurrency are not implied by this vocabulary.

The first bounded proof uses a one-field integer state and an integer result.
The method constructs a fresh state, returns it with the result, and leaves its
input receiver unchanged. Go uses a typed value receiver and generic
`Transition[S, R]`; JavaScript uses its native class/object method and an
immutable `{ state, result }` value. Both must lift to the same canonical graph,
and JavaScript projection must re-lift without drift.

The target ABI returns updated state and result together in canonical order.
Native Go, native JavaScript, standalone Wasm, and Pulp must agree across
positive, negative, and overflow vectors. Certification must reject receiver
type disagreement, method non-membership, argument mismatch, malformed
transitions, hidden recursion, and invalid request/response layouts.

Starting with v26, executable program envelopes are linked against the
independently pinned Core Execution schema module instead of copying every
schema declaration into every program. Program values still carry the same
stable schema identities. This keeps canonical programs bounded, removes
duplicate catalogue data, and preserves every self-contained v2-v25 artifact
byte-for-byte. Identical shared type declarations are deduplicated during
linking; conflicting declarations remain duplicated so canonical compilation
fails closed rather than selecting one silently.
