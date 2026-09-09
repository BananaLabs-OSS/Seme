# Core Execution Semantics v34

Version 34 adds `SliceConstruct`, a language-neutral immutable slice
constructor. It references an explicit `SliceType` and contains an ordered
list of zero through 512 element expressions.

Elements retain their declared order. Construction produces a fresh, dense
slice and neither aliases nor mutates source storage. This revision specifies
no source-language syntax, storage layout, or implementation strategy.
