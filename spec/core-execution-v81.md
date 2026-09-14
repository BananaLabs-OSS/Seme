# Core Execution v81: scoped type declarations

Core Execution v81 adds a language-neutral `ScopedTypeDeclaration` statement.
It places an existing semantic type identity into the lexical block where the
source language declared it. The identity and its fields remain canonical, but
projection must not hoist it into package or module scope.

The first Go realization supports local named record declarations. Go
projection emits the record at the same block position and suppresses a second
package-level declaration. Nearby aliases and unsupported local type forms
remain explicit native boundaries.
