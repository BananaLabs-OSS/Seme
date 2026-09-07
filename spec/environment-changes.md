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
