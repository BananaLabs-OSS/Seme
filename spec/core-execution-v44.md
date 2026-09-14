# Core Execution v44: typed native value flow

Core Execution v44 completes the first bounded flow of runtime-owned values
through ordinary Go locals. It does not import Go's runtime or type system into
neutral Core.

The Go provider can now:

- evaluate a native multi-result call once;
- retain its ordered `ProductType` even when one or more items are `NativeType`;
- project those typed items into local bindings; and
- invoke Go-owned pointer and interface methods through `NativeMethodInvocation`.

For example, `os.Stat` remains a Go-native invocation returning an ordered
product of `io/fs.FileInfo` and `error`. Seme can bind both results, perform the
explicit native error nil test, and invoke `FileInfo.IsDir` without claiming
that filesystem metadata, Go interfaces, pointer mechanics, or dynamic method
dispatch are universal language semantics.

Unsupported local application records remain unsupported. The bridge is
available only when the value originates from an attributable native operation
or an already typed native boundary.

The module schema is additive-compatible with v43; v44 freezes the completed
provider contract and a distinct reproducible module identity.
