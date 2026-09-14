# Core Execution v52: explicit discarded evaluation

Core Execution v52 preserves Go's blank-identifier assignment as an explicit
evaluation when it has the single form `_ = expression`.

The expression is evaluated exactly once and its result is discarded. It does
not create a local binding and does not erase native calls or effects hidden
behind the expression. This corrects a classification error where `_` was
previously treated as a missing declaration object. No schema is added beyond
v51.
