# Core Execution v2

Revision 2 adds canonical BooleanType and signed IntegerLessEqual semantics to
the revision 1 integer-addition core. The canonical interpreter remains under
`modules/execution/v1` because the same implementation executes both revisions
and reproduces from its S1 source. See `spec/core-execution-v2.md`.
