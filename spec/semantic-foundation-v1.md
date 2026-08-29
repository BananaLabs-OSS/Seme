# Semantic Foundation Module v1

Semantic Foundation v1 is the first versioned module above frozen Kernel v1.
It gives structural Kernel entities generic schema meaning without adding new
wire tags or trusted decoder concepts.

## Declarations

A module declares stable identities for schemas and fields. A schema has a
positive version and an ordered set of field declarations. A field declaration
contains:

- stable field identity;
- value shape;
- cardinality: `one`, `optional`, or `many`;
- first schema version in which it exists.

Value shapes recursively use Kernel value kinds: unit, boolean, unsigned,
signed, bytes, reference, list, record, and hole. Reference shapes may constrain
the referenced schema identity. Lists declare one element shape. Records name a
schema rather than duplicating its fields.

## Validation

Validation is deterministic in module, schema, entity, field, and list order.
It checks:

1. declaration identities are unique and ordered;
2. schema versions and `since_version` values are positive;
3. every declared field belongs to exactly one schema;
4. required fields occur once, optional fields at most once, and many fields
   contain a list whose elements match the declared element shape;
5. known fields match their recursive value shapes;
6. local references resolve before schema constraints run;
7. a known schema version is validated using declarations active at that
   version;
8. unknown fields and unknown future schema versions are preserved but not
   semantically certified;
9. unavailable imported schemas are preserved with an `unavailable` result;
10. malformed known semantics reject with a stable diagnostic.

The result for each entity is `certified`, `preserved_unknown`, `unavailable`,
or `rejected`. Preservation is not certification.

## Imports

An import declaration records a module identity, minimum revision identity, and
locally addressable imported schema identities. Resolution is an explicit input
to validation. Missing imports do not corrupt or discard their entities; those
entities remain unavailable and cannot satisfy operations requiring certified
semantics.

## Evolution

Adding an optional field or a new schema version is compatible when older
validators preserve it. Removing or changing existing field meaning requires a
new schema version and a declared migration. Identity is never reused.

The checked canonical module graph is authoritative. Host implementations are
differential oracles and ecosystem adapters only.
