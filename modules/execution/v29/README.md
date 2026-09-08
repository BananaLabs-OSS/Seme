# Core Execution Semantics v29

Version 29 adds mutable closure environments without hidden mutation. Capture
reads and updates are explicit, closure bodies sequence updates before results,
and stateful indirect calls return the updated callable state alongside the
user-visible result through the existing state-transition semantics.
