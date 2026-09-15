# Core Execution Semantics v108

Core Execution v108 retains typed Go conversions whose targets are composite or
qualified type expressions. This covers real application shapes such as
`[]byte(text)`, pointer conversions, and package-qualified named conversions
without mistaking them for ordinary calls.

No neutral schema changes. Run `./scripts/check-execution-v108.sh` to reproduce
the module and fixture evidence.
