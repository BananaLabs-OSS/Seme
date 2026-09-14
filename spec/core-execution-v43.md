# Core Execution v43: typed native field observation

Core Execution v43 adds `NativeFieldRead` and `NativeBindingRead`, explicit observation boundaries
for a field whose storage and access mechanics belong to a source runtime.
It records the language, fully qualified field identity, receiver expression,
and canonical result type. A target must provide that realization or reject
placement; Seme does not reinterpret the native object as a universal record.

The bounded Go provider admits exported fields on types owned outside the
package being lifted when the observed result already has a supported neutral
type. The initial real-project evidence includes `*net/http.Request.Method`
and `*net/http.Request.Host`. Local unsupported records are not relabeled as
native fields.

`NativeBindingRead` records a language, fully qualified binding identity, and
canonical result type for runtime-owned package/global storage. It does not
turn implicit global state into a neutral constant or canonical state cell.

The Go bridge also completes the native invocation boundary for procedures:
a native callable with no source-language result receives neutral `UnitType`.
The call remains an attributable native operation; only its absence of a
returned value is portable.

Native ordered products may contain explicit `NativeType` items alongside
neutral items. The Go provider uses this for results such as
`(os.FileInfo, error)`: the call and runtime-owned value stay Go-native while
canonical surrounding code can project and observe the independently typed
error item.

| ID | Name |
| --- | --- |
| `a072` | `NativeFieldRead` |
| `a0720` | `native_field_read.language` |
| `a0721` | `native_field_read.field` |
| `a0722` | `native_field_read.receiver` |
| `a0723` | `native_field_read.result_type` |
| `a073` | `NativeBindingRead` |
| `a0730` | `native_binding_read.language` |
| `a0731` | `native_binding_read.binding` |
| `a0732` | `native_binding_read.result_type` |
