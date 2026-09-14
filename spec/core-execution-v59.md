# Core Execution v59: typed native indexed range

Core Execution v59 expresses ordinary Go array and slice `range` statements
through canonical Seme places, sequencing, and while control flow. The ranged
expression is evaluated exactly once. Go's native collection type, `int` index,
element type, indexing, length, comparison, and assignment mechanics remain
explicit typed realization boundaries.

This supports application-owned and dependency-owned element types without
pretending that they are neutral Core values. General Go index expressions and
compound assignments receive the same typed treatment when their mechanics are
not already neutral.

Map range and string range deliberately remain native islands: map iteration
order and UTF-8 rune iteration are distinct mechanics that this milestone does
not approximate. No schema is added beyond v58.
