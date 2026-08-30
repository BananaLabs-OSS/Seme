# Provider Contract v1

## Purpose

A Seme provider connects an ordinary ecosystem project to canonical Seme
without making that ecosystem part of the trusted Kernel. The first Go
provider is a proof of this contract, not a claim of general Go support.

The contract covers one closed loop:

```text
native project -> ingest -> semantic patch -> project -> native validation
               -> re-ingest -> semantic equivalence
```

## Canonical module

Provider Contract v1 is canonical module `...7000`, initial revision `...7001`.
Its stable schema identities are:

| Identity | Schema |
|---|---|
| `...7010` | ProviderProfile |
| `...7011` | NativeFile |
| `...7012` | SourceOccurrence |
| `...7013` | Declaration |
| `...7014` | IdentityEvidence |
| `...7015` | OpaqueRegion |
| `...7016` | IngestionResult |
| `...7017` | ProjectionReport |

The checked declarations and complete field identities are authoritative in
`modules/provider/v1/module.seme`. A provider's host manifest is operational
evidence; the emitted Kernel envelope is canonical semantic authority.

## Provider declaration

Every provider invocation declares:

- contract version and provider identity;
- provider implementation version;
- language, language version, toolchain, target, and environment profile;
- supported construct set and explicit exclusions;
- project revision or native content digests;
- fidelity for every imported semantic entity or opaque region.

Support is always a claim about this complete profile. A provider must not
claim unqualified support for a language.

## Ingestion result

An ingestion result contains:

- a deterministic revision digest;
- stable semantic entity identities;
- native provenance and source occurrences;
- semantic shape understood by the provider;
- `exact`, `refined`, `adapted`, `emulated`, `embedded`, `native`, or
  `impossible` fidelity;
- preserved opaque regions and diagnostics;
- the native file digest against which projection is valid.

Existing identities are input to re-ingestion. A provider must recover an
identity when its declared matching evidence remains valid. It must report an
ambiguity instead of silently assigning an old identity to a different entity.

## Semantic patches

A patch names its base revision and target identities. Operations carry
preconditions over the semantic state they expect. Applying a patch is atomic:
all operations validate and project, or native and canonical state remain
unchanged.

Provider Contract v1 initially requires one operation:

```text
RenameDeclaration(entity, expected_name, replacement_name)
```

The contract will grow from proven operations rather than anticipating every
language edit.

## Projection

Projection must:

1. reject a stale base revision or changed native file digest;
2. modify only source occurrences proven to resolve to the target entity;
3. preserve all other bytes unless a declared native formatter contract changes
   them;
4. write atomically;
5. run the declared native validation command when requested;
6. emit a report of changed files, ranges, and validation results.

Opaque source is never regenerated merely because it surrounds an understood
entity.

## Reconciliation states

Each managed project or region is in one explicit state:

- `synchronized`;
- `native_changed`;
- `canonical_changed`;
- `both_changed`;
- `conflicted`;
- `opaque`.

Provider v1 only projects from `synchronized` with a patch based on the current
revision. Other states fail closed pending a later reconciliation contract.

## Round-trip conformance

A conforming fixture proves:

- the native project works before import;
- ingestion assigns deterministic semantic identities;
- a semantic rename changes only resolved occurrences;
- the native formatter and tests pass;
- re-ingestion recovers the renamed declaration's identity;
- unaffected entity identities and semantics are unchanged;
- removing Seme artifacts leaves an ordinary native project.

Passing this contract demonstrates a provider boundary. It does not establish
complete support for the provider's language.

## Implementation boundary

Providers should use their ecosystem's maintained parser, resolver, type
checker, formatter, build system, package manager, compiler, and test runner.
Provider code owns translation, identity evidence, fidelity classification,
patch projection, and reconciliation. Reimplementing mature language machinery
inside a provider requires separate justification and conformance evidence.

Provider Contract v1 is intentionally specified independently of a host SDK.
The Go proof supplies the first implementation. A shared SDK should be
extracted only after a second provider demonstrates which code is independent
of Go; the Kernel must not absorb accidental assumptions from either adapter.

## First Go conformance profile

The v1 proof profile is deliberately narrow: ordinary packages within one
module, package-level functions, resolved definition/call occurrences, no generated
files, no cgo, no methods, no build-tag variants, and one changed native file
per projected transaction. It uses the installed Go parser, type checker,
module command, and test command.

The proof executes the rename through canonical Patch Module v1 first. The Go
provider accepts only a validated committed graph whose sole semantic entity
change is the target declaration's `declaration.name` field. It then edits only
the recorded resolved occurrences, syntax-checks the result, validates an
isolated project copy, and atomically replaces the single declared file.

Run `./scripts/check-provider-v1.sh` for deterministic import, canonical
Kernel/Foundation validation, Patch execution, fail-closed stale/uncommitted
cases, native tests, re-import, identity recovery, semantic equivalence, opaque
preservation, and canonical projection-report evidence.

The native revision covers the recursive module source closure: root and
nested `.go` files plus `go.mod` and `go.sum`, excluding `.git` and `vendor`.
V1 projects package-level function declarations from every source package in
the module closure. Target analysis resolves and type-checks a directly
imported local helper using those package-qualified identities. Reachability
filtering and general expression lifting remain later expansions.
