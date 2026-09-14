# Core Execution v56: typed native assignment materialization

Core Execution v56 preserves type-checked Go assignments whose target type is
not yet a neutral Core type.

The right-hand expression remains canonical or native according to its own
meaning. Its Go assignment conversion or materialization is then represented as
an explicit typed native operation. This covers inferred `int`, application
records, maps, and other Go-owned locals without pretending their mechanics are
neutral or changing evaluation count. No schema is added beyond v55.
