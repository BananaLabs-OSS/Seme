# Core Execution v62: ignored parallel results

Core Execution v62 extends the Go provider's compositional control lifting for
parallel and multi-result assignments that contain Go's blank identifier.

Every right-hand side is still evaluated exactly once in source order. A blank
target discards only its result; it does not discard the evaluation or its
effects. When every target is blank, the product-valued call remains one
evaluated statement rather than creating an unusable synthetic local.

This is Go-provider syntax and mechanics over existing neutral Seme evaluation,
product, local, and assignment semantics. It adds no Go concept to the neutral
core and no application-specific meaning.
