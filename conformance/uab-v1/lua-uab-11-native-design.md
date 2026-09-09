# Lua UAB-11 native cumulative application

The Lua oracle uses ordinary Lua 5.1/LuaJIT functions plus explicit sealed Seme
value adapters. `policy.lua` and `application.lua` are separate source modules.
The application validates missing keys, index bounds, and negative elements
before constructing state or requesting effects. It then modular-sums the
values through a mutable capture, dynamically selects offset or scale policy,
adds an immutably captured amount, immutably replaces and appends the adjusted
value, updates the selected counter, and returns an explicit successful
transition followed by exactly one authorized `observability.log(true)`.

The native gate includes fixed offset/scale/error/capability cases and 128
independent, deterministic sequences of 16 state-threaded commands: 2,048
observations in total. The shared xorshift corpus includes modular boundaries,
successful transitions, and all three rejection paths.

This is explicitly an **unscored intermediate checkpoint**. Generic Lua v35
support now also covers modular multiplication, Option-returning map lookup,
scalar protocol witnesses, runtime-selected scalar dynamic dispatch, projection,
and byte-identical re-lift. The complete `Apply` function still requires generic
nested record-field expressions, Option/Result callback blocks, traversal with
early typed rejection, record construction, and nested transition/result
construction. Canonical application execution, Wasm/Pulp parity, full
projection, adversarial rejection, and cross-language identity are therefore
not claimed here.
