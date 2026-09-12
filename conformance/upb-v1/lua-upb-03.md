# Lua UPB-03 evidence

Status: claimed 2026-09-12 by `scripts/check-lua-upb-03.sh`.

The cumulative gate passed all seven evidence classes. An ordinary two-module
Lua project declared one exact external rock dependency. The Lua resolver
authenticated the lock selection and complete external tree, represented both
the local and ecosystem dependencies through neutral Dependency v1, and bound
the result into Project v4.

Two offline resolutions, Dependency-v1 artifacts, and Project-v4 compositions
were byte-identical. Canonical projection retained the native dependency files
exactly and the projected project independently reproduced the same closure.
The inherited generated corpus agreed through original/projected Neovim Lua,
canonical evaluation, standalone Wasm, and pinned Pulp.

Floating or malformed requirements, altered roots or sources, unknown lock
fields, unsafe tree paths, external byte drift, dependency-authority tampering,
and destination collisions rejected without partial replacement.

The run used Node.js, Neovim, and cached Go 1.26.0 through process-local
environment variables. No software was installed and no persistent setting
changed.
