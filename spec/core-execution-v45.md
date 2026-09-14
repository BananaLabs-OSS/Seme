# Core Execution v45: attributable native default values

Core Execution v45 adds `NativeDefaultValue`, an explicit realization boundary
for a source language's default initialization semantics.

The operation contains:

- the owning language; and
- the exact `NativeType` being initialized.

The Go provider uses this operation for ordinary local `var` declarations when
their type is owned by Go. Neutral booleans, integers, strings, byte sequences,
supported slices, maps, and canonical records continue to use their existing
neutral constructors. Go-specific structs, interfaces, pointers, and other
runtime-owned values are not silently assigned a universal Seme zero value.

This distinction preserves exact Go behavior while allowing a function to mix
neutral control flow with explicitly native initialization and method calls.
Other language providers may define their own default realization without
inheriting Go's rules.
