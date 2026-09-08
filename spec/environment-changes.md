# Environment changes

This file records development-environment changes made while advancing Seme.
It is not part of Seme's runtime or trusted bootstrap.

## 2026-09-07

- No software or language toolchain was installed.
- Repository-local Git author settings were restored from the existing commit
  history so verified work could be committed:
  - `user.name = SirNiklas`
  - `user.email = 253184012+SirNiklas9@users.noreply.github.com`
- No global Git settings were changed.
- Go validation ran on the existing workstation toolchain over SSH.
- The downstream acceptance harness uses `GOWORK=off` per command to isolate
  its small Go module from a platform-specific parent workspace. This is not a
  persistent environment setting.
- Core v12 validation uses the existing Go, Node, and Pulp installations on the
  workstation. No dependency or persistent setting was added.
- The semantic-module gate requires the existing GNU `timeout` command. Each
  subprocess has a 120-second limit by default; a single invocation may change
  that limit with the `SEME_SEMANTIC_STEP_TIMEOUT` environment variable. This
  is a per-command setting, not a persistent environment change, and no package
  was installed for it.
- The generic Pulp proof runner was made capability-free in Pulp commit
  `acc66ca`; this is a source change, not an environment change.
- Syncthing may create transient `.syncthing.*.tmp` siblings while transferring
  checked artifacts from the workstation. That pattern is now ignored so a
  synchronization race cannot accidentally stage transport files. Four such
  files briefly entered an unpushed local commit; that commit was amended, its
  reflog expired, and its object pruned. The temporary files were removed.
- Live Language Service v1, the incremental Go session, and the certified
  canonical builder used only the existing workstation tools documented above.
  No software was installed and no persistent environment setting changed.

## 2026-09-07 — temporary Go toolchain cache for Core v14 verification

The pre-existing local `mise` Go shim automatically downloaded Go 1.26.0 into
the task-scoped temporary cache directories `/tmp/seme-v14-gomodcache` and
`/tmp/seme-v14-gocache` because the repository requires Go 1.26. No persistent
configuration, project dependency, system package, or workstation toolchain was
installed or changed. The v14 Pulp gate also downloaded its already-locked Go
module dependencies (`BurntSushi/toml`, `vmihailenco/msgpack/v5`, and
`tetratelabs/wazero`, including transitive locked modules) into the same
task-scoped module cache; repository dependency declarations were unchanged.
These exact temporary directories were removed after the verification run.

The Core v14 gate sets `XDG_CACHE_HOME` to its automatically removed temporary
work directory for the duration of the command. This lets the existing Pulp
Wazero runtime create its compilation cache without writing to the developer's
persistent home cache. It is not a persistent environment setting.

## 2026-09-08 — pinned JavaScript parser dependency

The JavaScript reference provider added repository-local package metadata and
the exact dependency `acorn@8.15.0`. It was installed with npm into the ignored
`reference/js/node_modules/` working directory; no global package or system
setting changed. `package-lock.json` records the registry artifact and
integrity digest so clean environments can reproduce the parser dependency.

## 2026-09-08 — Core v15 lexical-local verification

Core v15 used the existing workstation Go 1.26 toolchain, Node 22 runtime,
pinned Acorn dependency, Pulp checkout, and Seme bootstrap. No package,
toolchain, global dependency, or persistent setting was installed or changed.
All Go, Pulp, and Wasm caches used by the gate are scoped to its automatically
removed temporary directory where applicable.

## 2026-09-08 — Core v16 multi-function verification

Core v16 used the existing workstation Go 1.26 toolchain, Node 22 runtime,
repository-local pinned Acorn dependency, existing Pulp checkout, and Seme
bootstrap. No package, toolchain, dependency, global setting, SSH setting, or
persistent workstation configuration was installed or changed. Temporary gate
artifacts and caches are scoped to automatically removed task directories.

## 2026-09-08 — Core v17 compositional-record verification

Core v17 used only the existing workstation Go 1.26 toolchain, Node 22,
repository-local Acorn dependency, Seme bootstrap, and pinned Pulp source. No
software, dependency, global setting, SSH setting, or persistent environment
configuration was installed or changed. Gate artifacts use an automatically
removed temporary directory.

## 2026-09-08 — Core v18 mutable-place verification

Core v18 used only the existing workstation Go 1.26 toolchain, Node 22,
repository-local Acorn dependency, Seme bootstrap, and pinned Pulp source. No
software, dependency, global setting, SSH setting, or persistent environment
configuration was installed or changed. Gate artifacts and caches use an
automatically removed temporary directory.

## 2026-09-08 — Core v19 structured-choice verification

Core v19 used only the existing workstation Go 1.26 toolchain, Node 22,
repository-local Acorn dependency, Seme bootstrap, and pinned Pulp source. No
software, dependency, global setting, SSH setting, or persistent environment
configuration was installed or changed. A temporary generator copy under
`/tmp` avoided a Syncthing race; it did not alter workstation configuration.

## 2026-09-08 — Core v20 effect verification

Core v20 used only the existing workstation Go 1.26 toolchain, Node 22,
repository-local Acorn dependency, Seme bootstrap, and pinned Pulp source. The
gate copies a Seme-owned capability adapter into an automatically removed
archived Pulp worktree before compilation; neither the Pulp repository nor its
history is modified. No software, dependency, global setting, SSH setting, or
persistent environment configuration was installed or changed.

## 2026-09-08 — Core v21 fixed-array verification

Core v21 used only the existing workstation Go 1.26 toolchain, Node 22,
repository-local Acorn dependency, Seme bootstrap, and pinned Pulp source. New
module artifacts were generated on the workstation and copied into the Seme
repository after checksum verification. The follow-on fixed-array boundary
gate reused those same tools and an automatically removed archived Pulp tree.
No software, dependency, global setting, SSH setting, or persistent environment
configuration was installed or changed.

## 2026-09-08 — Core v22 fold verification

Core v22 used only the existing workstation Go 1.26 toolchain, Node 22,
repository-local Acorn dependency, Seme bootstrap, and pinned Pulp source. Its
gate builds only in automatically removed temporary directories. No software,
dependency, global setting, SSH setting, or persistent environment
configuration was installed or changed.

## 2026-09-08 — Core v23 runtime-slice verification

Core v23 used only the existing workstation Go 1.26 toolchain, Node 22,
repository-local Acorn dependency, Seme bootstrap, and pinned Pulp source. Its
gate builds and executes only in automatically removed temporary directories.
No software, dependency, global setting, SSH setting, or persistent environment
configuration was installed or changed.
