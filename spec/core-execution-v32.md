# Core Execution v32: total variant matching and opaque bytes

Core Execution v32 adds provider-neutral elimination for Result and Option
values and the minimum byte-value operations required by UAB-v1.

`VariantBinding` names one typed payload available only inside its selected
match arm. `VariantBindingRead` observes that payload. It is a lexical semantic
binding, not a pointer, storage slot, mutable variable, tuple position, or
source-language pattern.

`ResultMatch` contains a Result value and exactly two total branch blocks. Its
success branch receives the success payload binding; its error branch receives
the error payload binding. Exactly one branch is evaluated.

`OptionMatch` contains an Option value, one total absence branch block, and one
total presence branch block whose payload is available through a binding.
Exactly one branch is evaluated. Absence has no hidden payload.

Both match nodes require their branch blocks to produce the same surrounding
expected type. Binding type agreement, lexical scope, selected constructor,
and branch result agreement are certification obligations; target layout is
not part of these schemas.

`BytesLiteral` carries an exact, immutable sequence of octets. `BytesEqual`
compares length and every octet and returns Boolean. Bytes have no text
encoding, termination, host address, ownership, capacity, or storage identity.

## Frozen identities

| Suffix | Declaration |
|---|---|
| `a060` | `VariantBinding` |
| `a061` | `VariantBindingRead` |
| `a062` | `ResultMatch` |
| `a063` | `OptionMatch` |
| `a064` | `BytesLiteral` |
| `a065` | `BytesEqual` |

Their fields occupy the corresponding suffix ranges `a0600` through `a0651`
as emitted by the normative module artifact. Version 32 is strictly additive
over v31.
