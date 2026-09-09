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

## 2026-09-08 — Core v24 collection-query verification

Core v24 used only the existing workstation Go 1.26 toolchain, Node 22,
repository-local Acorn dependency, Seme bootstrap, and pinned Pulp source. Its
gate builds and executes only in automatically removed temporary directories.
No software, dependency, global setting, SSH setting, or persistent environment
configuration was installed or changed.

## 2026-09-08 — Core v25 immutable-collection verification

Core v25 used only existing Go and Node toolchains, the repository-local Acorn
dependency, Seme bootstrap, and pinned Pulp source. During workstation network
unavailability, generation and focused checks used the existing local Go 1.25.6
toolchain with an automatically removed temporary module-file copy; the
repository module declaration and machine configuration were not changed. The
acceptance gate is specified to use the existing workstation Go 1.26 toolchain
and builds only in automatically removed temporary directories. At this
checkpoint the workstation was unreachable at the network layer, so the full
gate and cumulative v20-v25 gates were instead run in an automatically removed
local copy with only its copied Go version declarations changed to 1.25. No
software, dependency, global setting, SSH setting, or persistent environment
configuration was installed or changed.

## 2026-09-08 — Core v26 method-transition verification

Core v26 used only existing Go and Node toolchains, the repository-local Acorn
dependency, Seme bootstrap, and pinned Pulp source. The workstation remained
unreachable at the network layer, so the complete gate ran in an automatically
removed local copy using the existing Go 1.25.6 toolchain; only copied Go
version declarations were adjusted from 1.26 to 1.25. No repository dependency,
software, global setting, SSH setting, or persistent environment configuration
was installed or changed.

## 2026-09-08 — Core v27 interface-dispatch verification

Core v27 used the existing workstation Go 1.26 toolchain, Node 22,
repository-local Acorn dependency, Seme bootstrap, and pinned Pulp source. The
complete acceptance gate builds and executes only in an automatically removed
temporary directory. Local investigation used the existing Go 1.25.6
toolchain and disposable `/tmp` copies. No software, dependency, global
setting, SSH setting, or persistent environment configuration was installed or
changed.

## 2026-09-08 — Core v28 immutable-closure verification

Core v28 used the existing workstation Go 1.26 toolchain, Node 22,
repository-local Acorn dependency, Seme bootstrap, and pinned Pulp source. Its
complete acceptance gate builds and executes only in an automatically removed
temporary directory. No software, dependency, global setting, SSH setting, or
persistent environment configuration was installed or changed.

## 2026-09-08 — Core v29 mutable-closure verification

Core v29 used the existing workstation Go 1.26 toolchain, Node 22,
repository-local Acorn dependency, Seme bootstrap, and pinned Pulp source. Its
complete acceptance gate builds and executes only in an automatically removed
temporary directory. No software, dependency, global setting, SSH setting, or
persistent environment configuration was installed or changed.

## 2026-09-09 — Core v30 runtime-map verification

Core v30 used the existing workstation Go 1.26 toolchain, Node 22,
repository-local Acorn dependency, Seme bootstrap, and pinned Pulp source. Its
complete acceptance gate builds and executes only in an automatically removed
temporary directory. No software, dependency, global setting, SSH setting, or
persistent environment configuration was installed or changed.

## 2026-09-09 — Core v30 cumulative composition verification

The cumulative Core gate used the existing workstation Go 1.26 toolchain,
Node 22, repository-local Acorn dependency, Seme bootstrap, and pinned Pulp
source. Verification ran in an automatically removed temporary copy; the
repository, Pulp checkout, and workstation configuration were not modified.
No software, dependency, global setting, SSH setting, or persistent environment
configuration was installed or changed.

## 2026-09-09 — UAB-v1 UAB-01 language bridges

Go and JavaScript UAB-01 verification used the existing workstation Go 1.26
and Node 22 toolchains, Seme bootstrap, repository-local dependencies, and
pinned Pulp source. Lua verification reused Neovim 0.11.2's existing embedded
LuaJIT 2.1 runtime with all XDG data, state, and cache paths redirected to an
automatically removed temporary directory. An initial local Go build could not
download Go 1.26 because of the restricted environment and made no change; the
official gates were rerun successfully with the existing workstation Go 1.26
toolchain. No software, dependency, global setting, SSH setting, or persistent
environment configuration was installed or changed.

## 2026-09-09 — Core v31/v32 and UAB-02 foundations

Core v31/v32 generation and preservation checks used the existing local Go
1.25.6 fallback with a disposable module file and cache because the authorized
Go 1.26 workstation was unreachable at the network layer. Every v2-v32 module
regenerated byte-identically; v31/v32 hashes and canonical compilation passed.
The composite reactor executed under existing Node 22 and pinned Pulp source
using temporary caches. Lua compound tests reused the existing Neovim/LuaJIT
runtime with temporary XDG paths. A blocked automatic Go 1.26 download wrote
nothing. No software, dependency, global setting, SSH setting, or persistent
environment configuration was installed or changed.

## 2026-09-09 — Lua UAB-03 through UAB-11 verification

Lua UAB-03 through UAB-11 used the existing Node 22 runtime, Go 1.25.6
toolchain, Neovim 0.11.2 embedded LuaJIT 2.1 runtime, Seme bootstrap, pinned
Pulp source at `acc66ca61fe69c5f2c4093bc55e13aeac6dcc001`, and the already pinned
repository-local `acorn@8.15.0` dependency. Each gate created an automatically
removed `mktemp` directory under `${TMPDIR:-/tmp}`; Go and runtime caches were
redirected there with `GOCACHE` and `XDG_CACHE_HOME`, and Neovim data, state,
and cache paths were redirected with `XDG_DATA_HOME`, `XDG_STATE_HOME`, and
`XDG_CACHE_HOME`. No timeout override was set. No software, dependency, global
setting, SSH setting, or persistent environment configuration was installed
or changed.

## 2026-09-09 — Core v33 through v35 reproduction

Core v33 through v35 generation and reproduction used the existing Go 1.25.6
toolchain and Seme bootstrap. Generation output and `GOCACHE` lived in each
gate's automatically removed `${TMPDIR:-/tmp}` directory. The v35 gate
regenerated v2 through v35 canonical module artifacts and compared them with
the checked artifacts. No timeout override was set, and no software,
dependency, global setting, SSH setting, or persistent environment
configuration was installed or changed.

## 2026-09-09 — shared cross-language UAB-12 verification

UAB-12 used the existing Go 1.25.6 toolchain, Node 22 runtime, Neovim
0.11.2/LuaJIT 2.1 runtime, Seme bootstrap, pinned Pulp source, and the existing
repository-local Acorn dependency. Its generated Go, JavaScript, Lua,
canonical, Wasm, and Pulp evidence and all tool caches were confined to an
automatically removed `${TMPDIR:-/tmp}` directory; `GOCACHE`,
`XDG_CACHE_HOME`, and Neovim's XDG data/state/cache paths were redirected into
that directory. No timeout override was set. No software, dependency, global
setting, SSH setting, or persistent environment configuration was installed
or changed.

## 2026-09-09 — UAB-v1 completion audit

The completion audit reused the existing toolchains and dependencies above.
Read-only default cache paths were avoided with process-local `GOCACHE` and
XDG paths under `/tmp`. The semantic-module audit was rerun with the documented
process-local `SEME_SEMANTIC_STEP_TIMEOUT` first set to 300 seconds after its
120-second default expired and then to 900 seconds when one validator exceeded
300 seconds; neither value persisted beyond its command. No
software, dependency, global setting, SSH setting, or persistent environment
configuration was installed or changed.

## 2026-09-09 — Useful Project Bridge v1 foundation

UPB-v1 profile, Project Contract v1 generation, typed ProjectSnapshot
validation, canonical wire encoding, and the semantic-module registry reused
the existing Go 1.25.6 toolchain, Node 22 runtime, Seme bootstrap, and
repository-local semantic
validators. Project-contract gates create an automatically removed
`mktemp` directory under `${TMPDIR:-/tmp}` and redirect `GOCACHE` into it;
additional focused verification used a process-local cache under `/tmp`.
The composed instance-validator work also ran the existing Go race detector
with a separate process-local `GOCACHE` under `/tmp`; it installed nothing and
did not alter Go, repository, shell, SSH, or system configuration.
Go Project Build v1 reused the checked-in frozen K0 bootstrap and compiler,
created automatically removed staging directories under `/tmp`, and published
only create-new artifacts inside those temporary gate directories. Its native
fixture used the already installed Go 1.25.6 toolchain with a temporary
`GOCACHE`. No compiler, dependency, service, shell setting, or system package
was installed or changed.
No software, dependency, global setting, SSH setting, or persistent
environment configuration was installed or changed.

## 2026-09-09 — Project Contract v2 source-inventory foundation

Project Contract v2 generation, deterministic source discovery, and exact
round-trip staging, canonical inventory validation, and detached bundle tests
reused the existing Go toolchains and Seme bootstrap.
Focused unit and race tests redirected `GOCACHE` to disposable directories
under `/tmp`; the contract reproduction gate used an automatically removed
`mktemp` directory. No software, dependency, global setting, SSH setting, or
persistent environment configuration was installed or changed.

## 2026-09-09 — Go UPB-01 foundation gate

The Go UPB-01 foundation reused Go 1.25.6, Node 22, the checked Seme
bootstrap, repository-local dependencies, and pinned Pulp source at
`acc66ca61fe69c5f2c4093bc55e13aeac6dcc001`. Native, canonical, standalone
Wasm, and Pulp runs plus all caches and copied Pulp sources lived in an
automatically removed directory under `/tmp`. No software, dependency, global
setting, SSH setting, or persistent environment configuration was installed
or changed.

The detached projected-tree materializer added immediately afterward used the
same existing Go toolchain with a disposable `/tmp` `GOCACHE`; it installed or
changed nothing outside repository source and temporary test directories.
