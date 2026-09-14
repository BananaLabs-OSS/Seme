# Core Execution v71: typed native slicing

Core Execution v71 preserves language-owned two-index and full slicing as an
explicit typed expression with independently optional low, high, and maximum
bounds. Go realizes native slice syntax; JavaScript and Lua expose explicit
adapters without claiming identical bounds or memory behavior.

Go projection realizes supported `make`, `append`, `delete`, `copy`, `new`,
`cap`, `clear`, and `len` invocations as native Go syntax. JavaScript and Lua
views expose the invocation's language, callable, signature, ordered arguments,
and result type through explicit runtime adapters. This makes mixed canonical
and native expressions visible without claiming that foreign runtimes share
Go's collection mechanics.
