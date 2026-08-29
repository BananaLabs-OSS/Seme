# Reference oracles

Code under `reference/` is non-authoritative differential machinery. It makes
module rules executable early, supplies conformance cases, and helps ecosystem
providers integrate without placing a host language in Seme's trusted path.

The authoritative forms are checked canonical Seme module graphs. A reference
oracle is removed from a freeze claim unless its behavior is reproduced by the
canonical implementation and compared against the same fixtures.

The initial Go oracle covers Semantic Foundation v1 shape/cardinality/version
validation and Patch Module v1 atomic rename transactions. It deliberately uses
the Go standard library rather than reimplementing Go language semantics.
