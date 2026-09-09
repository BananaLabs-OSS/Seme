# JavaScript UAB-09 candidate design

This bounded cell maps the explicit source form `console.log(boolean)` to the
neutral `observability.log` effect and capability. The mapping is **adapted**:
it does not claim JavaScript console formatting, coercion, host-object, or I/O
semantics. Other `console` methods and non-Boolean payloads remain unsupported.

The one-source corpus `Observe(first, second)` makes ordering observable by
recording `first`, then `second`, and returning `second`. Its shared vectors
cover every Boolean pair. The acceptance gate compares the exact named result
and ordered trace produced by original JavaScript, projected JavaScript,
canonical execution, standalone Wasm, and the pinned Pulp runtime.

Authorization is an execution input, not a source assumption. Canonical
execution pre-certifies the complete statement list before running any effect.
Standalone Wasm and Pulp denial are exercised for every shared vector and must
trap/fail with an empty trace. Structural tests reject forged effect names,
capability names/references, argument shape, and a forged later invocation
without allowing a partial earlier observation. A located source rejection
keeps unsupported `console.error` outside the cell.

This file documents candidate evidence only. It does not itself award or alter
the UAB score.
