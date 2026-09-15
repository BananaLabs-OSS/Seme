# Core Execution v96

Core Execution v96 preserves Go's native Unicode string-range mechanic.

The existing language-qualified `NativeRange` contract now accepts a Go string
collection and retains Go's exact `int` byte-index and `rune` code-point
bindings. This is intentionally native rather than pretending string iteration
is identical in every language. Go projection reconstructs ordinary
`for index, codepoint := range value` source.

v96 adds no schema identity; all v95 module declarations and identities remain
byte-for-byte unchanged.
