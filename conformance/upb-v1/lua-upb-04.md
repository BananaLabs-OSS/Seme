# Lua UPB-04 evidence

Status: claimed 2026-09-12 by `scripts/check-lua-upb-04.sh`.

The cumulative gate passed all seven evidence classes. An ordinary Lua root
module constructed `slice<i64>` and passed it through `require` into a function
owned and exported by a second module. Canonical function identities, call
edges, parameter/result types, module ownership, visibility, source origins,
and the pinned dependency closure were retained by the project authorities.

Canonical projection recreated the two-module Lua project and re-imported to
the exact original G1. Original and projected Neovim execution, canonical
evaluation, standalone Wasm, and pinned Pulp agreed on normal and signed i64
boundary inputs. Wrong value kinds and invalid collection indices rejected.

Inherited graph, lock, integrity, source, malformed authority, and publication
adversaries also passed without partial replacement. The initial unsupported
multi-statement Wasm-call shape was not claimed; the accepted cell explicitly
records its single-expression boundary.

The run used Node.js, Neovim, and cached Go 1.26.0 through process-local
environment variables. No software was installed and no persistent setting
changed.
