# Go UAB-12 shared projection convergence

The shared UAB-12 seed is the canonical application graph lifted from the
JavaScript UAB-11 application under package `seme.uab11/application`, entry
`Apply`, revision 1. Independent language implementations are not required to
produce the same internal algorithm: UAB-12 instead proves that every language
can open this one Seme application as ordinary native source and preserve its
exact meaning when that projection is re-imported.

Go projection carries a deterministic `seme:projection-v1` envelope containing
the exact canonical bytes and their SHA-256 digest. This is provenance, not a
trusted escape hatch. The filesystem adapter accepts those bytes only after:

1. parsing and type-checking the complete ordinary Go source through the normal
   source provider;
2. decoding the envelope and verifying its SHA-256 digest;
3. structurally validating the graph by projecting its typed program; and
4. comparing that independent projection byte-for-byte with the current source.

Thus formatting or semantic edits invalidate the envelope. A changed digest,
changed payload, truncated envelope, or a valid but different graph attached to
the old source also rejects. The user may remove the envelope and lift the
actual Go program normally, but it then receives identities derived from its
current semantics rather than falsely retaining the projected graph's identity.

The envelope is generic: it contains no application, fixture, function, or
package-name branch. Stable per-declaration and receiver directives remain
useful for normal source lifting, while the verified envelope preserves exact
expression and control identities where two languages necessarily spell the
same canonical operation with different surface constructs.
