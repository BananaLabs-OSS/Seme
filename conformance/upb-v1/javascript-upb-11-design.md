# JavaScript UPB-11 acceptance design

Status: implemented and claimed by `scripts/check-javascript-upb-11.sh` on
2026-09-11.

JavaScript UPB-11 applies the neutral Patch v1, Live Language Service v1, and
Project v14 contracts to the cumulative ES-module project. JavaScript parsing,
identifier validity, native snapshot hashing, import binding, JSON configuration
projection, and Node validation remain provider mechanics; none are added to
Seme Core or the neutral contracts.

A provider-owned session accepts only strictly increasing complete snapshots.
An invalid newer snapshot advances the observed revision while retaining the
last-valid canonical graph. A stale snapshot cannot replace accepted or
last-valid state. An edit is bound to the SHA-256 revision of every declared
JavaScript and structured-reference file that the editor observed.

The bounded semantic edit renames exported `InitializePolicy` to `BuildPolicy`.
It addresses the canonical declaration identity, updates its declaration, named
ES-module import, and typed configuration reference, preserves matching comment
prose, then runs native behavior validation. Re-lift uses explicit identity
evidence so the spelling changes while the canonical identity remains stable.

The full gate independently rebuilds both revisions through Project v12, emits
strict authenticated bundles, derives and publishes each Project-v13 target
placement, and binds their exact transition through Patch v1 and Project v14.
Two final publications must be byte-identical.

Stale revisions and native bytes, simultaneous native/semantic edits,
ambiguous declarations, replacement collisions, invalid JavaScript identifiers,
forged semantic identities, metadata tampering, and existing destinations must
reject without partial publication. Only `scripts/check-javascript-upb-11.sh`
may claim the cell, and it must rerun the complete UPB-10 predecessor plus the
focused and full-project proofs.

This bounded cell does not claim arbitrary JavaScript refactoring, a three-way
merge, broad npm/runtime support, native-island execution, or Workbench.
