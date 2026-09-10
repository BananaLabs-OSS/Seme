# Package Contract v3

Package v3 adds language-neutral annotations for canonical non-function
declarations already owned by v2 and ownership for concrete realizations that
have no source member. Records, interfaces,
receiver-associated callables, and concrete generic realizations retain their
Execution identities; this contract does not redefine their structure.

Generic definitions are referenced only when a corresponding canonical
definition entity exists. Source aliases are not preserved as semantic
declarations and may be expanded by projection.
