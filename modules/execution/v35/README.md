# Core Execution Semantics v35

Version 35 adds `MapLookupOption`, a language-neutral runtime-keyed map lookup
that distinguishes an absent key from a present value equal to the value
type's zero. It references the map expression, key expression, and the exact
canonical `OptionType` returned by the lookup.

Lookup does not prescribe hashing, allocation, mutation, exception behavior,
or source-language syntax. A present key produces `Some(value)` and an absent
key produces `None`.
