# Core Execution v46: symmetric typed native results

Core Execution v46 completes the first symmetric result-flow contract for
native Go invocations.

A native function or method may return:

- a neutral type;
- one explicit `NativeType`; or
- an ordered `ProductType` containing either kind.

Each result can enter an ordinary local binding while retaining its exact type
identity. The provider evaluates product-producing calls once and projects
items in source order. Pointer and interface methods use the same rule.

This release adds no schema beyond v45. It closes an adapter asymmetry; it does
not make runtime-owned types portable or assign them neutral behavior. The
language, target, signature, result type, and operation remain attributable in
the canonical graph.
