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

Run `./scripts/check-lua-provider-v1.sh` for the complete UAB-01 evidence gate.
The gate reuses the already-installed Neovim/LuaJIT runtime with all XDG state
redirected into its temporary directory; it installs and configures nothing.
