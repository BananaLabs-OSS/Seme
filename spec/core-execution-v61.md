# Core Execution v61: typed native defer

Core Execution v61 adds `NativeDefer(language, invocation)` as a first-class
control statement. For Go, evaluating the statement captures the invocation's
receiver, function value, and argument values immediately, then registers the
captured invocation on the current function's native defer stack. Registered
calls execute in last-in-first-out order whenever that function exits.

The node deliberately remains language-qualified. It does not claim that Lua,
JavaScript, or other cleanup constructs share Go's exact panic, recovery, named
result, and function-exit mechanics. Go projections emit native `defer`;
JavaScript and Lua views expose an explicit `Seme.deferNative` boundary whose
runtime must provide the declared Go mechanic before execution is available.

This milestone preserves defer as a typed native control island inside an
otherwise canonical function. It does not approximate defer as an immediate
call and does not claim a neutral cross-language cleanup semantic.
