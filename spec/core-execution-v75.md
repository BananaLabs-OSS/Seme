# Core Execution v75: compound-assignment mutability

Core Execution v75 makes the Go control provider recognize every compound
assignment operator during its mutability analysis. The corresponding update
was already modeled; v75 closes the earlier mismatch that incorrectly rejected
otherwise supported mutable locals.

No new canonical schema is required. Exact Go projection and existing explicit
JavaScript/Lua native-invocation views remain unchanged.
