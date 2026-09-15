# Core Execution Semantics v109

Core Execution v109 preserves contextually typed `nil` operands inside Go type
conversions such as `[]byte(nil)`. The nil remains an explicit Go native default
value while the enclosing conversion remains visible in canonical Seme.

No neutral schema changes. Run `./scripts/check-execution-v109.sh` to reproduce
the module and fixture evidence.
