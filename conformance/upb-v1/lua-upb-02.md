# Lua UPB-02 evidence

Status: claimed 2026-09-12 by `scripts/check-lua-upb-02.sh`.

The authoritative cumulative gate passed all seven evidence classes. It lifted
an ordinary two-module Lua project directly, derived a deterministic Package-v2
graph with explicit ownership, public/private visibility, `require` resolution,
export binding, source provenance, and typed canonical function signatures,
then committed that graph through Project v3.

Projection recreated both native Lua files rather than flattening the package.
The original and projected projects passed the generated boundary corpus under
Neovim and the projected files re-lifted to byte-identical canonical G1.
Canonical evaluation, standalone Wasm, and the pinned Pulp realization agreed
with native Lua, including signed 64-bit overflow behavior and malformed-call
rejection.

Repeated graphs and projects were byte deterministic. Missing modules, private
member access, import cycles, duplicate exports, source digest drift, malformed
graphs, and publication collisions rejected without replacing valid output.

The run used Node.js, Neovim, and the cached Go 1.26.0 toolchain through
process-local environment variables. No software was installed and no
persistent environment setting changed.
