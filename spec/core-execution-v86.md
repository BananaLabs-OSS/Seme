# Core Execution v86: typed native two-result map lookup

Core Execution v86 preserves an arbitrary Go map lookup with its value and
presence results as one typed native operation. The resulting product is bound
once and projected into ordinary Seme locals, so concurrent map state cannot be
observed through two inconsistent lookups.

The neutral `map[i64]i64` model remains unchanged. Map types outside that model
retain their exact Go type identities and mechanics as an explicit native island;
only the surrounding binding and control flow become canonical. No canonical
schema change is required because typed native invocation and product projection
already express the boundary.

Go's blank identifier discards the corresponding product projection without
removing or duplicating the lookup operation.
