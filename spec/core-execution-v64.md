# Core Execution v64: mutable parameter places

Core Execution v64 gives ordinary parameter reassignment a language-neutral
realization. When a supported function parameter is reassigned, the Go provider
creates a canonical mutable local place initialized from that parameter. Every
subsequent read and write uses the place.

For example, `value = strings.TrimSpace(value)` becomes a mutable binding whose
initializer reads `value`, whose assignment stores the call result, and whose
later return reads the updated place. This preserves evaluation order and
mutation semantics while allowing each projector to choose valid local syntax.

The change is version-gated. Older execution modules retain their prior
acceptance boundary. Go named result parameters, unsupported parameter types,
and language-specific aliasing remain outside this milestone rather than being
silently approximated.
