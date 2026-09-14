# Core Execution v91

Core Execution v91 preserves reads of imported Go package variables as typed
native bindings.

The canonical graph records the native language, complete package-and-binding
identity, and result type. Go projection restores the qualified binding and its
import. Execution remains authoritative in the declared native Go island; Seme
does not pretend that process-global runtime objects are portable values.

No canonical schema changed. Older module artifacts and identities remain
byte-for-byte reproducible.
