# Core Execution v41: neutral products

Core Execution v41 adds ordered heterogeneous product values and typed
projection. Products model the language-neutral meaning shared by constructs
such as Go multiple results, JavaScript arrays used as fixed tuples, and Lua
multiple returns. They do not adopt any source language's tuple mechanics.

`ProductType` records the ordered type of every item. `ProductProject`
evaluates its product once and selects one statically typed item by zero-based
index. Providers must reject arity, index, or item-type mismatches rather than
silently adapting them.

The initial Go bridge admits two through sixteen supported result items and
lowers a local multi-result call into one product evaluation followed by typed
projections. Existing Go comma-ok operations retain their established neutral
Option semantics.

`NativeType` preserves a runtime-owned type without pretending that its
mechanics are universal. The initial bounded use is Go's predeclared `error`:
it may appear in an otherwise typed product or native call, but remains a Go
runtime value until an explicit cross-language error adapter is selected.
Native variadic calls with explicit arguments retain their declared signature
and ordered arguments exactly. Source-level slice expansion remains unsupported
because it requires an additional call-shape contract.

| ID | Name |
| --- | --- |
| `a06f` | `ProductType` |
| `a06f0` | `product.item_types` |
| `a070` | `ProductProject` |
| `a0700` | `product_project.product` |
| `a0701` | `product_project.type` |
| `a0702` | `product_project.index` |
| `a0703` | `product_project.item_type` |
| `a071` | `NativeType` |
| `a0710` | `native_type.language` |
| `a0711` | `native_type.spelling` |
