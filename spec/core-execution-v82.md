# Core Execution v82: type-checked local constants

Core Execution v82 accepts local Go constant declarations after the Go type
checker has resolved every named constant. Constant uses lift as their exact
canonical values; declaration-only syntax has no runtime identity and is not
copied into the semantic graph.

This supports individual and grouped constants, inherited expressions, and
`iota` values within the already supported canonical value domains. Unsupported
constant value domains remain explicit at their use sites. No canonical schema
change is required.
