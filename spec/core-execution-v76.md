# Core Execution v76: native-aware scoped bindings

Core Execution v76 applies the existing native-aware type resolver consistently
to multi-result bindings and single- or multi-value `if` initializers. Product
items retain their individual type identities and scoped bindings remain
explicit in the canonical graph.

No new canonical schema is required. Exact Go projection and the existing
explicit JavaScript/Lua native type views remain unchanged.
