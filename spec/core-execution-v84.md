# Core Execution v84: nonterminal else-if chains

Core Execution v84 lifts a nonterminal Go `else if` chain by normalizing it to
the language-neutral structure `else { if ... }`. Conditions, lexical scope,
mutations, and subsequent statements retain their existing canonical meaning.

This normalization matches the same nested conditional concept used by
JavaScript and Lua and requires no canonical schema change.
