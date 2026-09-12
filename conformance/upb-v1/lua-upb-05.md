# Lua UPB-05 evidence

Status: claimed 2026-09-12 by `scripts/check-lua-upb-05.sh`.

The cumulative gate passed all seven evidence classes. The configuration plan
resolved three typed defaults, ran one validator, and executed the ordered
`configuration → policy → application` lifecycle. It produced a successful
runtime record with `Ready=true`, `Total=16`, and `Namespace="seme"` through
the neutral configuration executor.

Project v8 authenticated the complete Execution-v36 program, Package-v4 graph,
dependency closure, source inventory, and Configuration-v3 plan. Repeated
artifacts were byte-identical. Canonical projection restored all three ordinary
Lua modules, record declarations, exports, and functions and re-lifted to exact
canonical bytes.

Original Lua, canonical evaluation, the general canonical Wasm VM, and pinned
Pulp agreed on enabled, disabled, negative, overflow, and malformed inputs.
Invalid initialization edges, ambient environment reads, source/graph/
dependency tampering, and publication collisions rejected without partial
output.

The run used Node.js, Neovim, and cached Go 1.26.0 through process-local
environment variables. No software was installed and no persistent setting
changed.
