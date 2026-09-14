# Core Execution v68: native built-in invocation views

Core Execution v68 widens the existing `NativeInvocation` contract rather than
inventing neutral meanings for language-specific built-ins. The Go provider
now preserves `len` over Go-owned collection types, including when nested in
other typed built-ins such as `make`.

Go projection realizes supported `make`, `append`, `delete`, `copy`, `new`,
`cap`, `clear`, and `len` invocations as native Go syntax. JavaScript and Lua
views expose the invocation's language, callable, signature, ordered arguments,
and result type through explicit runtime adapters. This makes mixed canonical
and native expressions visible without claiming that foreign runtimes share
Go's collection mechanics.
