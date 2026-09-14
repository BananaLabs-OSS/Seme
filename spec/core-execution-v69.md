# Core Execution v69: local native field reads

Core Execution v69 retains typed field reads from locally declared Go-owned
structures. Surrounding collection and registry logic can remain canonical
while unsupported structures stay explicit native types.

Go projection realizes supported `make`, `append`, `delete`, `copy`, `new`,
`cap`, `clear`, and `len` invocations as native Go syntax. JavaScript and Lua
views expose the invocation's language, callable, signature, ordered arguments,
and result type through explicit runtime adapters. This makes mixed canonical
and native expressions visible without claiming that foreign runtimes share
Go's collection mechanics.
