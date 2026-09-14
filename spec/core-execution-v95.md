# Core Execution v95

Core Execution v95 preserves assignment through a typed native pointer.

The statement records the realization language, canonical pointer expression,
and canonical assigned value. Pointer mutation stays an explicit mechanic rather
than being mislabeled as pure universal state. Go projection reconstructs
ordinary `*target = value` behavior.

v95 adds `NativeDereferenceAssignment`; all prior schema identities and module
artifacts remain unchanged.
