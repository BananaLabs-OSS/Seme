# Kernel v1

Kernel v1 is the frozen structural substrate shared by the self-hosted compiler
and every semantic module. It deliberately does not interpret domain schemas.

The canonical representation is specified in `kernel-wire-v1.md`. The semantic
foundation built on it is specified separately in `kernel-meta-v1.md`; that
module may grow without changing or reopening Kernel v1.

## Kernel concepts

```text
Identity      opaque stable identity allocated once
Revision      immutable content revision with parent reference
Entity        identity + schema + version + fields
Reference     typed edge to another stable identity
Hole          structurally identified unresolved semantic value
Module        identity of the module owning an envelope
```

Constraints, effects, capabilities, provenance, refinements, diagnostics,
memory, concurrency, objects, and other meanings are versioned schemas and
modules represented through these concepts. They are Seme semantics, but not
hard-coded Kernel wire semantics.

## Required execution substrate

The first kernel reader/executor needs primitive integers, booleans, bytes,
lists, records, functions, calls, locals, structured control flow, and explicit
effects for arguments and atomic filesystem operations.

These are bootstrap execution facilities. They do not establish a universal
object, memory, concurrency, or error model.

## Freeze evidence

Kernel v1 is frozen because the checked implementation can:

1. represent its bootstrap schema and canonical validator;
2. preserve and structurally validate an external semantic module without
   kernel modification;
3. preserve and round-trip an unknown module field;
4. reject malformed wire, ordering, identities, local references, and reserved
   bootstrap relationships;
5. encode canonically with deterministic diagnostics;
6. execute or lower its own compiler;
7. migrate at least one deliberately obsolete fixture.

The executable gate is `../scripts/check-kernel-v1.sh`; frozen identities,
artifacts, and hashes are recorded in
`../compiler/KERNEL-V1-FREEZE.md`. Schema-level validation and migration belong
to their owning versioned semantic modules.

## Non-goals

Kernel v1 does not standardize a preferred source syntax, object system,
ownership model, garbage collector, exception model, scheduler, database,
package ecosystem, instruction set, or execution target. Those remain semantic
modules and target contracts so the kernel does not collapse every language
into one lowest-common-denominator worldview.
