# Core Execution v98

Core Execution v98 preserves classic Go `for` loops whose initializer binding
uses a native integer type such as `int`.

The initializer, comparison, mutable loop place, and increment/decrement retain
their exact Go mechanics and types. The loop body stays canonical and Go
projection reconstructs equivalent executable control flow.

v98 adds no schema identity; all v97 declarations and identities remain
unchanged.
