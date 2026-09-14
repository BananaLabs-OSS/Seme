# Core Execution v47: nested typed native field flow

Core Execution v47 lets a native field read produce another explicitly typed
native value, which a later native field read or invocation may consume.

For example, Go's `request.URL.Path` is retained as two attributable field
observations:

- `request.URL` produces the Go-owned `*net/url.URL` native type;
- `.Path` produces the neutral Seme string type.

This release adds no schema beyond v46. Runtime-owned intermediate values stay
native islands; neutral results rejoin canonical Seme. Local application types
cannot be mislabeled as external runtime boundaries.
