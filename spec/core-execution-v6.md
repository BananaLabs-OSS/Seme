# Core Execution Semantics v6

Core Execution v6 additively extends v5 with `FunctionCall` (`...9060`). A call
references one canonical `Function` and an ordered list of argument
expressions. Calling is semantic; inlining or a concrete machine call is a
lowering decision.
