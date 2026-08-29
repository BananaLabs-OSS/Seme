# Core Execution Semantics v2

Core Execution revision 2 (`...9002`, parent `...9001`) preserves revision 1
and adds:

| Identity | Schema |
|---|---|
| `...9020` | BooleanType |
| `...9021` | IntegerLessEqual |

Function result and Parameter type references now admit any canonical type;
each executable profile must validate the supported mechanics explicitly.
IntegerLessEqual carries left and right expressions plus their IntegerType.

The first v2 profile accepts exactly three signed modular i64 parameters and a
BooleanType result for `parameter + parameter <= parameter`. Signed comparison
is implemented by mapping two's-complement ordering into unsigned ordering; the
addition remains modular. The canonical S1 interpreter supports both revision 1
integer addition and revision 2 quota-policy graphs.

No control flow, records, allocation, effects, or arbitrary expressions are
implied by this revision.
