# Core Execution v77: predeclared native methods

Core Execution v77 recognizes methods whose authoritative source language
defines them without a package owner. The first bounded realization is Go's
predeclared `error.Error()` method. Seme retains the typed receiver, arguments,
result, signature, and explicit `builtin` ownership of that Go mechanic.

The Go projector realizes the existing `NativeMethodInvocation` schema as an
ordinary receiver method call. JavaScript and Lua continue to expose the call
as an explicit Go native island rather than claiming equivalent mechanics.

No new canonical schema is required.
