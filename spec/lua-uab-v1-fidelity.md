# Lua UAB-v1 fidelity profile

## Runtime pin

The initial direct bridge targets the Lua 5.1 language subset implemented by
LuaJIT 2.1.1741730670 embedded in Neovim 0.11.2 on Linux amd64. This pin is an
acceptance environment, not a dependency of canonical Seme execution. The
bridge installs or configures nothing.

## UAB-02 representation decisions

Lua spelling alone does not distinguish the required canonical value families.
The source boundary therefore uses explicit annotations and will reject an
unclassified table, number, string, or `nil`.

| Canonical family | Planned Lua boundary | Fidelity and unresolved work |
|---|---|---|
| Boolean | Lua `boolean` | Exact. Only `false` and `true`; truthy values are not coerced. |
| signed i64 | `seme.i64` annotated LuaJIT `int64_t` cdata | Adapted. Ordinary Lua 5.1 numbers are binary64 and cannot represent all i64 values. Arithmetic and comparison require checked cdata-specific evidence; numbers must not silently lift. |
| text | `seme.text` annotated Lua string | Refined. Lua strings are arbitrary bytes; the bridge boundary must validate canonical UTF-8 scalar text. Byte length is not character length. |
| bytes | `seme.bytes` annotated Lua string | Exact for immutable byte observations. Text and bytes remain distinct canonical types despite sharing a Lua representation. |
| records | Sealed, field-named tables with a declared record annotation | Adapted. Shape, missing fields, extra fields, mutation, aliasing, and metatables must be checked explicitly. |
| Result/Option | Tagged sealed tables | Adapted. Bare `nil` is not Option-none because assignment of `nil` deletes table entries and absent arguments affect Lua's arity observations. |
| fixed arrays | Sealed dense tables with declared length | Adapted from Lua's one-based indexing. Holes, non-integer keys, `#` ambiguity, mutation, and metatables reject. |
| slices | Sealed `{ length, values }` abstraction | Adapted. Raw tables cannot supply deterministic length or immutable update semantics. |
| runtime-keyed maps | Sealed entry abstraction with canonical key equality and deterministic traversal | Adapted. Raw table iteration order is unspecified, numeric key equality differs across representations, and `nil` deletes entries. |

The provider currently implements the distinct Boolean, i64, text, and bytes
type identities for typed identity/call/return functions. This is structural
lift/projection groundwork only. UAB-02 remains incomplete until runtime
boundary adapters, all compound families, native/canonical/target parity, and
nearby rejection evidence pass together.

`reference/lua/seme-values.lua` now supplies the repository-owned immutable
runtime representation foundation for the compound side of this mapping. It
uses protected userdata handles and private weak storage so a raw table or
`nil` is never mistaken for a canonical value. It provides explicit tagged
Option and Result values, zero-indexed fixed-array/slice observations, ordered
record fields, range-checked decimal i64 handles, distinct text/byte handles,
and runtime maps sorted by canonical key tokens. This is native adapter
evidence, not yet provider or target evidence for the compound families.

The direct provider and projector now map `seme.option<T>` and
`seme.result<T,E>` annotations plus `Seme.some`, `Seme.none`, `Seme.ok`, and
`Seme.err` calls to the neutral Core v31 Option and existing Result schemas.
Constructors are contextually typed; bare `nil`, wrong-arm values, and
unrecognized wrapper calls reject. `scripts/check-lua-compound-v1.sh` compiles,
validates, projects, and byte-equivalently re-lifts representative Option and
Result graphs.

Explicit `seme.array<T,N>`, `seme.slice<T>`, and `seme.map<K,V>` annotations
now lift to the existing neutral fixed-array, slice, and runtime-map type
schemas and project without drift. Raw `table` remains rejected. Slice
construction is deliberately unavailable because Core has no standalone
slice-literal schema; supported fixed-array and map operations use their
existing neutral construction and update paths.
Declared `---@class` records with ordered `---@field` entries now lift to
neutral `RecordType` and `RecordField` entities and project without drift.
Raw table shape is never inferred. `Seme.field(record, "name")` maps to the
neutral `FieldRead` schema with declared result-type checking and projects
without drift. Record construction remains unclaimed.

`Seme.array(...)`, `Seme.length`, and explicitly zero-based
`Seme.index_zero` now map to `FixedArrayConstruct`, `CollectionLength`, and
`IndexRead`. `Seme.empty_map`, explicitly zero-on-missing `Seme.lookup_zero`,
and `Seme.map_update` map to `EmptyMap`, `MapLookup`, and immutable `MapUpdate`.
Argument, element, key, value, result, arity, and fixed-index bounds are checked
before graph emission. Slice construction remains intentionally absent because
Core provides no slice-literal schema.

Empty-map and zero-lookup source spellings carry an explicit `"i64"`,
`"text"`, or `"bytes"` value descriptor. The native wrapper retains that
descriptor even when the map has no entries and constructs the corresponding
exact canonical zero on a miss. A missing or disagreeing descriptor rejects.
Other value types remain unsupported for zero-on-missing lookup until their
canonical zero construction is specified. Fold is also deferred: an honest
mapping requires Lua callback lifting plus canonical accumulator and iteration
bindings, not a host-side wrapper loop.

The bounded total expression
`Option<Result<bytes,text>> -> Boolean` now lifts directly from structural Lua
callback syntax to `OptionMatch`, `ResultMatch`, scoped `VariantBinding` reads,
`BytesEqual`, and text equality. It projects and re-lifts without drift, and
its five native vectors agree through deterministic Wasm and pinned Pulp.
Partial matches and out-of-scope binding shapes reject rather than receiving
guessed meaning. This is the exact v32 composite profile, not general pattern
matching or general composite lowering.

## Hard blockers that must not be guessed

- Lua 5.1 `number` cannot be called canonical i64. The full range requires an
  explicit LuaJIT cdata representation or a separately specified integer
  object.
- A Lua string cannot be inferred as text or bytes. The declared boundary and
  UTF-8 validation determine the mapping.
- `#table` is not a general collection length operation in the presence of
  holes, and `pairs` order cannot define canonical traversal.
- Raw `nil` cannot simultaneously mean absent value, deleted map entry, missing
  record field, and a canonical Option-none.
- Metatable behavior and identity-sensitive table keys remain outside the
  canonical collection mappings unless isolated behind an explicit adapter.
