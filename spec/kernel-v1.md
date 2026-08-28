# Kernel v1 draft

Kernel v1 is deliberately not frozen yet. A construct belongs here only if the
self-hosted compiler and every semantic module require it.

## Required entities

```text
Identity      opaque stable identity allocated once
Revision      immutable content revision with parent reference
Entity        identity + schema + version + fields
Reference     typed edge to another stable identity
Constraint    deterministic predicate over semantic values
Hole          required but unresolved semantic decision
Effect        declared interaction with an external authority
Capability    permission required to perform an effect
Provenance    author/tool/policy and originating revision
Refinement    evidence-bearing relationship between semantic levels
Module        versioned collection of schemas, rules, and operations
Diagnostic    stable rule ID + semantic location + structured arguments
```

## Required execution substrate

The first kernel reader/executor needs primitive integers, booleans, bytes,
lists, records, functions, calls, locals, structured control flow, and explicit
effects for arguments and atomic filesystem operations.

These are bootstrap execution facilities. They do not establish a universal
object, memory, concurrency, or error model.

## Freeze gates

Kernel v1 may be frozen only after it can:

1. represent its own schema and validator;
2. represent one external semantic module without kernel modification;
3. preserve and round-trip an unknown module field;
4. reject malformed references, schemas, effects, and control flow;
5. encode canonically with deterministic diagnostics;
6. execute or lower its own compiler;
7. migrate at least one deliberately obsolete fixture.

