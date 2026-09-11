# JavaScript UPB-08 acceptance design

Status: accepted implementation design; unclaimed.

JavaScript UPB-08 extends the cumulative Project-v10 authority with native
transport command/event records and two pure functions selected as dispatch and
replay. Ordered Transport v1 remains language-neutral: it owns stream, kind,
sequence, correlation, replay, frame-bound, codec, capability, receive, and send
meaning. JavaScript supplies ordinary declarations that are resolved by exact
semantic identity and authenticated package ownership.

The pure dispatch behavior must agree across native JavaScript, canonical Seme,
standalone Wasm, and pinned Pulp, including signed-i64 wrapping and malformed
record rejection. Stateful framing/transport execution remains a separately
authenticated host boundary. Pinned Pulp is only a pure synchronous carrier;
it is not claimed as the authoritative Ordered Transport port.

Project v11 composition must reproduce from the complete authenticated
v8→v9→v10 chain. Unknown fields, missing owners/functions, mixed dispatch and
replay, duplicate identities, stale prior artifacts, and output collisions must
reject without partial publication. The cumulative UPB07 projection must retain
native `transport.js` and re-lift the complete project graph exactly.

Only `scripts/check-javascript-upb-08.sh` may claim this cell.
