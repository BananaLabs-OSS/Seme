# JavaScript UPB-09 acceptance design

Status: claimed 2026-09-11 by `scripts/check-javascript-upb-09.sh`.

JavaScript UPB-09 adds native record declarations for explicit clock samples,
seeded random state/draws, application commands/state/results, and pure next,
dispatch, and replay functions. The language-neutral Controlled Effects v1
adapter must resolve every declaration from the authenticated Project-v11
graph with exact ownership and signatures.

Clock values are explicit replayable inputs; random evolution uses the frozen
signed-i64 modular `state*48271+1` algorithm; external Boolean effects are
logical requests requiring `observability.log`, not claims of delivery.
Ambient `Date`, host entropy, filesystem, network, or Pulp capabilities cannot
substitute. Native JavaScript, canonical Seme, standalone Wasm, and pinned Pulp
must agree on the pure controlled dispatch, including overflow and malformed
boundaries. Stateful effect delivery remains an explicit host placement.

Project v12 must bind a reproducible Controlled-Effects-v1 plan to the exact
Project-v11 chain. Policy/bound changes, missing declarations, signature or
ownership drift, mixed prior artifacts, unknown fields, and output collisions
must reject without partial publication. The cumulative projected JavaScript
project must preserve and re-lift `controlled.js` as ordinary native source.

Only `scripts/check-javascript-upb-09.sh` may claim this cell.
