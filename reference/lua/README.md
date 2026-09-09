# Direct Lua bridge

This directory begins Seme's direct Lua bridge. It does not translate Lua
through Go or JavaScript.

The pinned initial source profile is the common Lua 5.1 subset exercised by
Neovim 0.11.2's embedded LuaJIT 2.1.1741730670. The first bounded slice accepts
named local and global functions across multiple supplied files, identifier
parameters, direct calls, and returns. Every parameter and result must have an
EmmyLua-style `boolean` annotation. Projection emits ordinary Lua in that same
profile, and re-lifting must reproduce identical canonical Seme.

This is intentionally not general Lua support. Tables, numbers, strings,
multiple returns, varargs, assignments, branches, loops, closures, methods,
metatables, coroutines, module loading, environment mutation, errors, and
standard-library calls reject rather than receive guessed semantics. Source
files form one explicitly ordered lexical unit for this slice; Lua's package
loader is not modeled.

Lua's number, table, indexing, `nil`, multiple-return, metatable, environment,
and error semantics require explicit fidelity classifications before later UAB
cells can pass. No claim about those behaviors is made here.

Canonical Boolean is an adapted boundary, not Lua truthiness. Native evidence
passes values through `Seme.boolean`, which accepts only the Lua values `true`
and `false`; truthy numbers, strings, tables, and functions reject. Projection
retains ordinary annotated Lua Boolean parameters because the adaptation lives
at the invocation boundary rather than changing the function's meaning.

Run `./scripts/check-lua-provider-v1.sh` for the complete UAB-01 evidence gate.
The gate reuses the already-installed Neovim/LuaJIT runtime with all XDG state
redirected into its temporary directory; it installs and configures nothing.

The bounded UAB-02 aggregate proof uses explicit Seme-owned wrappers rather
than treating Lua tables, numbers, or `nil` as canonical values. Its structural
source profile composes a one-field i64 record read, a zero-based read from a
two-element fixed i64 array, an i64-slice length, and a lookup-with-zero from a
sorted i64-to-i64 map using modular i64 addition. The certified boundary uses
ordered inline record/array values plus bounded slice/map descriptors and
rejects gaps, trailing bytes, duplicate or non-increasing map keys, and counts
above 512. This is not a claim of arbitrary record or collection lowering.

Run `./scripts/check-lua-uab-02-aggregate.sh` for original/projected native Lua
observations, canonical validation and byte-identical re-lift, deterministic
Wasm execution, malformed rejection, and the pinned Pulp runtime check. It
reuses the repository bootstrap compiler, installed Node and Neovim/LuaJIT,
and a temporary archive of the already-present sibling Pulp repository. All
caches and Neovim XDG directories are temporary; nothing is installed and no
persistent setting is changed.
