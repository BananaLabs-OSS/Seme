# Core Execution Semantics v114

Core Execution v114 permits canonical immutable closures to construct and
return other canonical immutable closures. A nested closure explicitly captures
both the enclosing closure's parameters and any transitive outer captures it
uses. Each lexical layer owns distinct capture bindings; inner placeholders are
not accidentally rebound to an outer layer.

Go method expressions are also preserved as typed native values with an explicit
receiver type and result-function type. This allows higher-order callbacks to
retain native Go method-set behavior without pretending it is language-neutral.
Run `./scripts/check-execution-v114.sh` to reproduce the nested-closure module,
fixture, lift, and projection evidence.
