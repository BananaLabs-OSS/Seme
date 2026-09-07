# Core Execution v12 Freeze Audit

| Property | Evidence |
|---|---|
| No semantic churn | V12 emits the same ordered schema declarations as v11 under a new revision identity. |
| Canonical ABI derivation | Parameter offsets and request/response sizes come from canonical parameter indices and type references. |
| Generic lowering | Entry, body, return, expression root, parameter locals, and result type are resolved from graph identities rather than source names. |
| Certification | Wasm emission accepts only an opaque certificate produced after program membership, statement legality, typing, purity, reference-shape, cycle, and traversal-budget checks. Malformed-graph tests exercise every category. |
| Standalone execution | Node invokes the generated reactor for Boolean and i64 functions and checks exact response bytes. |
| Actual Pulp execution | Pulp loads the checked artifact and invokes it through the opaque provider-call boundary. |
| Rejection | Incorrect request lengths and Boolean bytes other than zero/one return explicit nonzero status. |
| Allocation lifetime | Zero/oversized/exhausted allocation rejects, LIFO free reclaims safely, forged near-u32-limit and foreign frees preserve state, and 20,000 calls return the bounded arena to its initial address. |
| Provenance binding | ABI JSON binds canonical identities and digest, artifact digest, target/provider/fidelity, and ordered field layouts. |
| Exact Pulp dependency | Gate requires commit `acc66ca...`, tree `a11daea...`, ancestry, and the capability-free runner digest before building. |
| Frozen predecessors | The v12 gate regenerates v2-v11 module constructions byte-for-byte. |
