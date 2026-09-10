# Configuration and Initialization Contract v1

This language-neutral contract declares typed configuration fields, static or
default resolutions, capability-backed runtime inputs, validation bindings,
initializer dependencies, and lifecycle transitions. It imports exact
Foundation v1, Execution v35, and Package v3 contracts.

The canonical graph is an immutable plan. Capability-backed values contain a
capability identity but never a runtime value or secret. Ambient environment
reads have no implicit meaning. Runtime outcomes remain Execution state and
are not embedded in the plan.
