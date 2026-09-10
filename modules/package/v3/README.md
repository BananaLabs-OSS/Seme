# Package Contract v3

Package v3 adds language-neutral ownership for canonical semantic declarations
that are not function members in the v2 detail graph. Records, interfaces,
receiver-associated callables, and concrete generic realizations retain their
Execution identities; this contract does not redefine their structure.

Generic definitions are referenced only when a corresponding canonical
definition entity exists. Source aliases are not preserved as semantic
declarations and may be expanded by projection.
