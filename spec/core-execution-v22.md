# Core Execution Semantics v22

Core Execution v22 adds language-neutral deterministic reduction:

- `IterationBinding` gives an accumulator or element a typed lexical identity.
- `IterationBindingRead` references that identity within its valid scope.
- `Fold` references a collection, initial value, accumulator binding, element
  binding, and body expression.

The collection's canonical order defines left-to-right iteration. Empty
collections produce the initial value without evaluating the body. Bindings are
scoped to the fold body and cannot alias each other.

The first adapters recognize an idiomatic bounded addition fold:

```go
total := int64(0)
for _, value := range values { total += value }
return total
```

```javascript
return values.reduce((total, value) => total + value, 0n);
```

Both forms lift to byte-identical canonical programs. JavaScript projection
emits native `reduce` and re-lifts without drift. Fixed arrays now permit length
zero, allowing the identity behavior to be exercised rather than assumed.

The certified Wasm realization validates collection type, both distinct i64
bindings, exact binding reads, the i64 initial value, and the supported addition
body before emission. It folds packed fixed-array parameters in declared order.
Native Go, native JavaScript, standalone Wasm, and Pulp agree for empty values,
signed values, and modular i64 overflow.

This profile does not yet claim arbitrary fold bodies, index-aware folds,
early termination, effects inside folds, nested collections, dynamic slices,
or a behaviorally noncommutative ordering proof. The canonical model supports
body evolution; the current target deliberately rejects shapes it cannot prove.
