# UAB-12 exact cross-language application proof

The shared application is the JavaScript UAB-11 source lifted as package
`seme.uab11/application`, entry `Apply`, revision 1 against Core Execution v35.
JavaScript is only the selected native authoring surface for this fixture. The
resulting canonical graph—not JavaScript execution—is the shared application.

The UAB-12 gate projects that one graph independently to JavaScript, Go, and
Lua. Every native projection must execute the same 2,048-command corpus, and
every provider must re-import its projection to the exact original canonical
bytes. Behavioral equivalence without canonical byte identity does not pass.

Go and Lua may carry a verified projection envelope in comments. The envelope
is accepted only when its digest is valid and independently projecting its
graph reproduces the complete native source byte-for-byte. Editing either the
source or envelope therefore rejects rather than silently restoring stale
meaning. This is lossless projection metadata, not a source-language runtime
dependency or a fixture/name-specific lowering rule.

The same seed graph and recursive Pure Value ABI execute through the canonical
interpreter, standalone Wasm, and Pulp pinned to
`acc66ca61fe69c5f2c4093bc55e13aeac6dcc001`. Nearby native constructs and
forged graph/envelope/ABI inputs must reject without guessed meaning, response,
or partial effect trace.
