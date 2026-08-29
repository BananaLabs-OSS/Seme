# K0 lowering module v1 candidate

This semantic module represents portable bootstrap programs as canonical Kernel
entities. K0 byte images are derived execution artifacts; this module graph is
the editable semantic source.

The module identity is `0000000000000000000000000000a000`. Its initial schema
identities are:

| Identity | Entity |
|---|---|
| `0000000000000000000000000000a010` | Program |
| `0000000000000000000000000000a011` | Function |
| `0000000000000000000000000000a012` | Instruction |

## Program fields

| Identity | Meaning | Shape |
|---|---|---|
| `0000000000000000000000000000a100` | profile | unsigned (`2` P1, `3` P2) |
| `0000000000000000000000000000a101` | capabilities | unsigned bit mask |
| `0000000000000000000000000000a102` | functions | ordered list(reference Function) |
| `0000000000000000000000000000a103` | entry | reference Function |

P1 requires capabilities to be zero. Function list order determines K0 function
indices, while references keep semantic identities stable when projections are
reformatted.

## Function fields

| Identity | Meaning | Shape |
|---|---|---|
| `0000000000000000000000000000a110` | parameter count | unsigned |
| `0000000000000000000000000000a111` | local count | unsigned |
| `0000000000000000000000000000a112` | instructions | ordered list(reference Instruction) |

## Instruction fields

| Identity | Meaning | Shape |
|---|---|---|
| `0000000000000000000000000000a120` | opcode | unsigned K0 opcode |
| `0000000000000000000000000000a121` | operand | optional unsigned operand |
| `0000000000000000000000000000a122` | target | optional reference Instruction |

Instructions are entities rather than anonymous records so branches target
stable semantic identities. The lowerer derives function indices, instruction
indices, code sizes, and branch byte witnesses. Those values never enter the
canonical graph.

This module does not make K0 canonical Seme semantics. It is the finite bridge
used to express and execute the self-hosting compiler before richer mechanics
and target modules exist.
