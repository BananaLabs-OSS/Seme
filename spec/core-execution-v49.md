# Core Execution v49: mixed records and scoped product initializers

Core Execution v49 expands Go lifting in two compositional directions:

- an application record may contain explicitly Go-owned field types while the
  record and its neutral fields retain canonical identities;
- a multi-result call in an `if` initializer is evaluated once, retained as an
  ordered product, and projected into branch-scoped bindings.

Omitted native fields in keyed record literals use the existing attributable
`NativeDefaultValue`. Local application types are never relabeled as external
runtime types. This release adds no schema beyond v48.
