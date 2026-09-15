# Core Execution Semantics v102

Core Execution v102 generalizes Go classic `for` lowering from one induction
variable to ordinary supported initialization and post statements. Multiple
bindings and parallel assignments retain Go's evaluate-before-assign behavior,
including index-pair updates such as `i, j = i+1, j-1`.

Initialization, condition, body, and post steps lower into existing canonical
bindings, expressions, assignments, and `While` control. Post steps are also
inserted before `continue` operations targeting the lowered loop. No new schema
is needed, so v102 preserves the complete v101 schema identity set.

Run `./scripts/check-execution-v102.sh` to reproduce the module, compiled
artifact, fixture lift, and native Go behavioral projection proof.
