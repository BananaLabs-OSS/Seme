# JavaScript UAB-01 evidence

`scripts/check-javascript-uab-01.sh` proves all five evidence classes for
the bounded JavaScript UAB-01 cell.

- **Lift:** independently parsed ECMAScript modules form a deterministic
  multi-source package with resolved relative imports and typed calls.
- **Native parity:** boundary and signed vectors execute in Node.
- **Target parity:** two independent Wasm lowerings are identical; standalone
  Wasm and pinned Pulp agree with native execution.
- **Projection round trip:** canonical Seme projects to executable native
  JavaScript and re-lifts to identical canonical bytes.
- **Rejection:** unsupported expressions, missing or host imports, duplicate
  paths, and escaping paths fail with located diagnostics.

This cell does not imply broad ECMAScript or Node compatibility. Its explicit
fidelity boundary is recorded in `spec/javascript-uab-v1-fidelity.md`.
