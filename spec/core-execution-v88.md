# Core Execution v88

Core Execution v88 preserves Go's contextually typed `nil` as an explicit
typed native default inside otherwise representable expressions.

The provider derives the exact expected Go type from function and method
parameters or assignment targets. Pointer, slice, map, channel, function, and
interface nil values therefore remain Go mechanics instead of being rewritten
as empty neutral values. Targets without the declared Go native-default
realization must reject placement.

No canonical schema changed. Older module artifacts and identities remain
byte-for-byte reproducible.
