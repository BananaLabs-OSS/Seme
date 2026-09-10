# Package Contract v3

Package Contract v3 extends v2 with complete, language-neutral ownership of
canonical semantic declarations needed for package-preserving projection.

`OwnedSemanticDeclaration` supplements—never duplicates—v2 function members.
Its declaration reference points to the authoritative Execution entity. It
records package ownership, semantic name, visibility, source origin, and the
already-declared imports used by cross-package references. Records, behavioral
interfaces, receiver-associated callables, generic definitions, and concrete
generic realizations are neutral declaration kinds.

A generic realization names a canonical generic definition only when that
definition exists in the Execution graph. Implementations must not invent a
definition from source-language syntax. Source aliases are outside this
semantic ownership graph and may be expanded during projection.

All ownership and import lists are canonical, sorted, unique, and included in
the domain-separated `CompletePackageGraph` content revision.
