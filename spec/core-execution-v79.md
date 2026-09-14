# Core Execution v79: nested returning guards

Core Execution v79 preserves a one-sided Go guard containing a return when the
guard occurs inside a loop or another block that does not itself require a
terminal return. The absent branch means ordinary fallthrough in that enclosing
block; it is not rejected as an incomplete function result.

No new canonical schema is required.
