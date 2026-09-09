# Package Contract v2

This directory contains the generated G1 construction projection and canonical
Seme graph for the additive Package Contract v2. It preserves Package v1 and
adds neutral package membership, visibility, import binding, and source-origin
declarations. Reproduce and validate it with `scripts/check-package-v2.sh`.

Instance validation and Project integration are intentionally separate future
steps; this contract does not make Project v1 or v2 source-aware by itself.
