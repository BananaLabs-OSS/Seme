# Live Language Service Contract v1

This contract defines incremental source-to-canonical communication without
depending on a source language, transport, user interface, or target runtime.
Its module identity is `0000000000000000000000000000d000`; revision v1 is
`0000000000000000000000000000d001`.

## Update identity and ordering

A `DocumentIdentity` contains an opaque stable key. An `UpdateRequest` carries
that identity, a monotonically increasing client revision, the complete content
bytes, and their SHA-256 digest. A receiver verifies the digest before lifting.

For a given document, a request is stale when its client revision is less than
or equal to the accepted `DocumentState.client_revision`. Stale requests are
rejected before lifting and cannot mutate content, diagnostics, mappings, or
the last-valid canonical revision. Revision numbers are scoped to one document;
they are not canonical revision identities and are not compared across
documents.

## Lift results

`LiftResult.disposition` is:

- `0`: accepted and canonically valid;
- `1`: accepted but invalid; or
- `2`: rejected as stale.

An accepted request advances the stored client revision and content digest.
When lifting succeeds, `canonical_revision` is present and becomes
`last_valid_revision`. When lifting fails, `canonical_revision` is absent and
the prior `last_valid_revision` is retained. A missing last-valid revision is
valid before the first successful lift. Canonical revision and semantic
identity byte strings are exactly 16 bytes.

Diagnostics contain stable code and severity values, a human-readable message,
and optional half-open byte offsets. When present, both offsets must satisfy
`start <= end <= len(content)`. A `SemanticSourceMapping` relates one canonical
semantic identity to a half-open source byte range and a role. Mappings are
evidence about the accepted content digest, never durable semantic identity
authority by themselves.

Results are snapshots: returned diagnostic and mapping collections cannot
alias mutable provider storage. Transport implementations may serialize the
schemas differently, but must preserve their values, ordering, and stale-update
rules exactly.

## Schema inventory

| Identity | Schema |
|---|---|
| `...d010` | `DocumentIdentity` |
| `...d011` | `UpdateRequest` |
| `...d012` | `LanguageDiagnostic` |
| `...d013` | `SemanticSourceMapping` |
| `...d014` | `LiftResult` |
| `...d015` | `DocumentState` |

The checked canonical module graph is authoritative. The Go state transition
is a differential executable oracle for ordering, digest, last-valid, and
defensive-copy behavior.
