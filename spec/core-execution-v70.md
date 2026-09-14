# Core Execution v70: typed native range

Core Execution v70 preserves language-owned iteration as an explicit typed
statement. Go map range keeps Go's iteration mechanics while its loop body
remains canonical and available for inspection and projection.

Go projection realizes supported `make`, `append`, `delete`, `copy`, `new`,
`cap`, `clear`, and `len` invocations as native Go syntax. JavaScript and Lua
views expose the invocation's language, callable, signature, ordered arguments,
and result type through explicit runtime adapters. This makes mixed canonical
and native expressions visible without claiming that foreign runtimes share
Go's collection mechanics.
