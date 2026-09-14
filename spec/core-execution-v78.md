# Core Execution v78: mixed product short declarations

Core Execution v78 preserves Go mixed short declarations where one
multi-result evaluation both defines new names and assigns existing mutable
places. The canonical graph retains one product-valued call followed by ordered
projections into distinct bindings and places.

The Go projector recognizes that canonical relationship and reconstructs a
single `:=` statement, including blank projections. Other language projections
retain the explicit product and state-update semantics rather than inheriting
Go's declaration syntax.

No new canonical schema is required.
