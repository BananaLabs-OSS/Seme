# Core Execution v1 freeze audit

Core Execution v1 is frozen only for the deliberately narrow independent
execution proof described in `core-execution-v1.md`.

| Property | Result | Evidence |
|---|---|---|
| Canonical vocabulary | pass | Generated G1 and compiled Seme reproduce byte-for-byte and pass Kernel/Foundation validation. |
| Provider exactness | pass | Go parser and type checker resolve the exact signature, return, operator, and parameter objects. |
| Stable identity | pass | The provider declaration identity becomes the canonical Function identity. |
| Language neutrality | pass | Lifted executable entities contain no Go AST, syntax, position, or runtime object. |
| Independent runtime | pass | S1-authored canonical interpreter lowers byte-identically and executes with no Go component in the runtime command. |
| Behavioral evidence | pass | Normal, negative, zero, and signed-overflow vectors match the ordinary Go contract. |
| Fail-closed scope | pass | Unsupported source bodies and malformed runtime arguments reject with status 65. |

This freeze does not certify arbitrary expressions, control flow, allocation,
effects, packages, goroutines, reflection, generics, interfaces, or any other Go
construct. Those require explicit semantic schemas, lift rules, conformance
evidence, and a new version/profile. The proof freezes the contract for adding
those forms without redesigning the Kernel or provider boundary.
