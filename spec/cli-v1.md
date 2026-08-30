# Seme CLI v1

The repository-root `seme` command is the first user-facing workflow for the
bounded Go application profile.

```text
./seme doctor
./seme build PATH/TO/GO/PROJECT [OUTPUT.wasm]
./seme run PATH/TO/GO/PROJECT CURRENT DELTA LIMIT [SUBJECT]
./seme inspect PATH/TO/GO/PROJECT
./seme audit PATH/TO/PROJECTS [OUTPUT.json]
./seme demo
```

`build` runs the native Go tests, imports the recursively tracked module,
resolves target fidelity, compiles and validates the canonical plan, and lowers
it to Wasm. It preserves these inspectable products under `PROJECT/.seme/`:

- `manifest.json`: provider source closure and identities;
- `provider.g1`: provider semantic projection;
- `plan.g1` and `plan.seme`: readable and canonical execution plans;
- `application.wasm`: default executable artifact;
- `report.json`: target, fidelity, adaptations, artifact path, and digest.

`.seme/` is build state and is ignored by Git. Ordinary Go sources are not
rewritten. Unsupported programs exit with status 65 and retain the stable
machine diagnostic while adding a human explanation for common profile gaps.
`inspect` prints the most recent successful build's machine-readable report.

`audit` discovers immediate project directories and build markers up to two
levels below them. Its JSON report records languages, module/build files,
available providers, execution strategy, and current fidelity without claiming
that native-island projects are fully lifted.

This CLI is usable for `seme-go-quota-v1`; it does not claim general Go support.
`./seme doctor` prints the exact supported surface.
