# Source Presentation Contract v1

Source Presentation v1 (`...1000@...1001`) preserves source-facing type-alias
intent without adding aliases to Core Execution meaning. A manifest is bound to
one exact Project-v9 snapshot and contains deterministic alias presentations
plus a 32-byte content revision.

Each alias names its owning Package-v4 package, stable source name, closed
package/public visibility, canonical target Execution type, authenticated
Project-v8 source unit and Package-v4 origin, and the Package-v4 import
bindings its spelling references. Raw source text and language-specific syntax
are not embedded. Providers may project native syntax only after validating
this metadata against the same authenticated project.

This contract imports Package v4, Execution v36, and Project v9 exactly. It
does not define parsing, type identity, package discovery, or rewrite policy.
