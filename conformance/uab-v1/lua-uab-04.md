# Lua UAB-04 evidence

Lua UAB-04 directly maps explicit, immutable collection construction, length,
checked zero-based indexing, traversal, append, update, removal, map lookup,
insertion, and removal. The Seme-owned Lua adapters preserve dense canonical
collections and deterministic traversal without claiming that ordinary
one-based Lua tables, `nil` deletion, coercion, or unordered table iteration
have those semantics.

`scripts/check-lua-uab-04-native.sh` derives its observations from one typed
vector corpus. Eleven focused entry points prove each operation independently.
A scalar-returning composed entry exercises Core v34 `SliceConstruct`, modular
folding, chained slice mutation and queries, and symbolic immutable map
insert/remove/lookup in one canonical graph. Map removal has an
outcome-distinguishing vector in which the inserted and removed key are equal.

Original Lua under Neovim's embedded LuaJIT, projected Lua, the structural
canonical evaluator, standalone Wasm, and Pulp pinned at
`acc66ca61fe69c5f2c4093bc55e13aeac6dcc001` produce identical named results.
Projection re-lifts byte-identically. Every composed malformed vector is
directly invoked through all five paths, including negative update indexing and
out-of-bounds removal. Source rejection tests target the function actually
mutated and reject raw tables, raw indexing, wrong index kinds, `nil` map
deletion, and raw Lua fold arithmetic.

Provider, projector, evaluator, and target behavior is selected by typed syntax
and canonical schema structure. No fixture, package, function, or vector name
selects semantics.
