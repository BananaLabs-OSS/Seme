# Lua UPB-02 acceptance design

Status: implemented and claimed by `scripts/check-lua-upb-02.sh` on
2026-09-12.

The bounded native project contains two ordinary Lua modules. Its root uses a
static `require("math.sum")`; each module publishes an explicit returned table,
and the dependency module retains both a public function and private helper.
These Lua mechanics belong to the provider. They normalize to existing
canonical calls while neutral Package v2 retains module ownership, imports,
exports, native visibility, source spans, and canonical signatures.

Project v3 authenticates the complete Package-v2 graph and Project-v2 source
inventory. Canonical projection combines semantic meaning with this graph to
recreate the two native modules, including `require`, private declarations, and
returned export tables. The regenerated project executes independently under
Neovim and re-lifts to the exact canonical bytes.

The sole gate reruns Lua UPB-01, graph/unit adversaries, deterministic neutral
project assembly, original/projected native parity, canonical evaluation,
standalone Wasm, and pinned Pulp. Missing or private imports, duplicate exports,
cycles, digest drift, malformed authority, and output collisions reject.

This cell does not claim dynamic `require`, arbitrary module loaders, LuaRocks,
global-name collisions across modules, or general Lua module compatibility.
