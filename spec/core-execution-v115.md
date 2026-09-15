# Core Execution Semantics v115

Core Execution v115 preserves a typed native lexical closure when a source
language's closure mechanics cannot yet be represented by neutral canonical
capture semantics. The native region records its language, complete function
signature, source realization, and result type. It remains an explicit native
mechanic inside an otherwise canonical function; it is not mislabeled as a
portable closure.

The first realization is Go. It covers recursive closures that share mutable
lexical cells, including captured slices and function cells. Go projection
restores the native function literal, while other targets must retain or adapt
the declared native boundary rather than silently changing its behavior.

Run `./scripts/check-execution-v115.sh` to reproduce the module, fixture,
provider, projector, canonical-validation, and exact native-execution evidence.
