# Ecosystem providers

Providers are optional adapters around mature ecosystem tooling. They do not
implement language parsing, name resolution, type checking, formatting, or
native validation when the owning ecosystem already provides those operations.

Every provider exposes the same initial command shape:

```text
<provider> import  native project [previous semantic state] -> ingestion JSON
<provider> rename  ingestion JSON + semantic selector        -> patch JSON
<provider> project ingestion JSON + patch JSON                -> native edits
<provider> verify  before + patch + re-ingestion              -> evidence
```

The JSON envelopes and behavior are governed by
[`Provider Contract v1`](../spec/provider-contract-v1.md). A new provider must:

1. declare a versioned language, toolchain, target, and environment profile;
2. delegate semantic analysis and native validation to its real ecosystem;
3. classify every claimed mapping with explicit fidelity;
4. reject or preserve unsupported constructs without approximation;
5. pass the same import/edit/project/validate/re-import identity proof;
6. keep provider-specific data out of the Kernel.

The Go proof intentionally keeps its adapter local until a second provider
reveals which implementation pieces are genuinely reusable. Extracting a host
SDK before that evidence would risk encoding Go assumptions as the universal
contract. The contract and command behavior are reusable now; shared code comes
after the second implementation.
