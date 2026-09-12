# Lua UPB-04 acceptance design

Status: implemented and claimed by `scripts/check-lua-upb-04.sh` on
2026-09-12.

The bounded project constructs a canonical `slice<i64>` in its root module and
passes that aggregate through a native `require("policy.collections")` binding
to a separately owned, explicitly exported function. The dependency validates
and reads the collection before returning an i64 result. Thus typed value and
call semantics cross the real package boundary without source flattening or an
intermediate language translation.

Package v2 owns both declarations and their exact canonical parameter/result
types; Project v3 binds the source/module graph and Project v4 binds its pinned
local-plus-rock dependency closure. Projection reconstructs both Lua modules,
their import/export relationship, and the aggregate signature, then re-lifts
to identical canonical bytes.

The sole gate inherits UPB-01 through UPB-03 and proves valid, boundary, wrong-
type, negative-index, and out-of-range observations through native and
projected Neovim Lua, canonical evaluation, standalone Wasm, and pinned Pulp.

This cell deliberately proves a single-expression cross-package aggregate
call. It does not claim arbitrary Lua values, ecosystem-package calls, or Wasm
inlining of multi-statement callees.
