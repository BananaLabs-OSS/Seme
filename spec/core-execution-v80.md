# Core Execution v80: scoped nonterminal guard initializers

Core Execution v80 preserves initializer declarations on one-sided returning
guards inside loops and other nonterminal blocks. The initializer binding is
visible to the condition and guarded body while remaining absent from the
enclosing fallthrough scope, matching Go lexical ownership.

No new canonical schema is required.
