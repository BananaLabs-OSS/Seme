# Core Execution Semantics v112

Core Execution v112 permits an otherwise mutable outer binding to become an
immutable closure capture when its last lexical mutation strictly precedes the
closure construction. Function-valued recursive cells remain mutable. This
preserves common build-then-callback patterns without weakening the explicit
mutable-capture boundary.

No neutral schema changes. Run `./scripts/check-execution-v112.sh` to reproduce
the module and fixture evidence.
