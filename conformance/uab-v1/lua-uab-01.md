# Lua UAB-01 evidence

`scripts/check-lua-provider-v1.sh` proves all five evidence classes for the
bounded direct Lua UAB-01 cell.

- **Lift:** annotated Boolean functions across supplied Lua files lift directly
  to a validated canonical Program with resolved calls.
- **Native parity:** true and false vectors execute in the pinned
  Neovim/LuaJIT profile for both original and projected source.
- **Target parity:** independent Wasm lowerings and ABIs are byte-identical,
  and standalone execution agrees for both Boolean vectors.
- **Projection round trip:** canonical Seme projects to ordinary Lua and
  re-lifts to identical canonical bytes.
- **Rejection:** an undeclared cross-file call fails at its source location.

Lua does not pass through Go or JavaScript. The exact supported and unsupported
Lua semantics are recorded in `reference/lua/README.md`.
