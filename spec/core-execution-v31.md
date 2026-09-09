# Core Execution v31: neutral optional values

Core Execution v31 adds an optional-value semantic family without assigning
meaning from any source language or runtime:

- `OptionType` identifies exactly one contained value type.
- `OptionNone` constructs absence at one declared `OptionType`.
- `OptionSome` constructs presence at one declared `OptionType` and carries
  exactly one value.

Presence and absence are explicit canonical values. They are not null
pointers, sentinel values, zero values, exceptions, result errors, omitted
fields, or host-language truthiness. An ecosystem bridge must declare how its
native spelling maps to these semantics and must reject mappings that erase an
observable distinction.

The v31 module is strictly additive over v30. It defines representation and
structural validation only. Matching, projection, execution ABI, storage, and
garbage-collection policy belong to later independently versioned work.

## Frozen identities

| Identity suffix | Declaration |
|---|---|
| `a050` | `OptionType` |
| `a0500` | `option.value_type` |
| `a051` | `OptionNone` |
| `a0510` | `option_none.type` |
| `a052` | `OptionSome` |
| `a0520` | `option_some.type` |
| `a0521` | `option_some.value` |

The constructor type fields are required references to `OptionType`. The
contained type and present value are required unrestricted semantic
references, allowing validation and later typed certification without baking
an ecosystem-specific type universe into Core.
