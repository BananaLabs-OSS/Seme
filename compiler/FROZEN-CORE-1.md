# Frozen Core 1: independent self-hosting handoff

Frozen Core 1 is the first independently self-hosted Seme compiler floor for
Linux amd64. From this boundary onward, compiler development on the frozen
platform requires no assembler, linker, Python, S0, S1, or external language
toolchain.

This is not a claim that Kernel v1 or the universal Seme platform is complete.
Structured semantic diagnostics, arbitrary entity storage order, imported
schema resolution, additional platform ignition, native-language projections,
packages, and mechanics remain work above this floor.

## Trusted boundary

The reproduction path trusts only:

- the documented amd64 ISA and Linux syscall ABI;
- `bootstrap/seme-k0-linux-amd64`, SHA-256
  `ce4a6a282d21a7e152f0015e3dd38ffa665727e2af3db8eb18da7310c5603cac`;
- the K0 P2 contract in `bootstrap/K0.md`;
- Kernel wire v1 and the K0 semantic module candidate specifications;
- ordinary operating-system process invocation and byte comparison.

The earlier seed, A0, S0, S1, assembly sources, and their checked artifacts are
retained as auditable bootstrap history. They are not invoked below.

## Canonical authorities and derived tools

| Tool | Canonical graph SHA-256 | Derived K0 SHA-256 |
|---|---|---|
| semantic lowerer | `7c006d9e0d70e282717eee49f0c491b42565034cb82b216af4f8119a101cdfc4` | `dfd62f219bd99f220df853af6f512c23b2b6f0e4f923dc452c5c068574913efe` |
| G1 projection compiler | `c161cdbfb15a40ea8ee9504fa5c8efc5725128c4e4a72697f866dc07e37722d6` | `a7d7d2ba4266a80a0539690801a6acc225f8492a7c72083f7515989c2b6b2239` |
| Kernel wire validator | `10b3082e4664bc300cae7559a23fa9abdd763889ecd2323a0a3a07455e36d7d4` | `1388413720bbdd28d599f3bd1158a589d10b3323742ec8710607906a600f0309` |
| v0 migration tool | `b1a51b52cf987a00fc7b46a1f2bceaff0205590a3a023fcd286f39b3372aaa00` | `a1bdf160b9c2e89eefcad687eb0304cb82abe069dcc5b31856813bd8e7a3da4b` |

The `.seme` graph is canonical. The matching `.g1` file is its checked readable
projection with explicit stable identities. The `.k0` file is derived.

## Reproduction

Starting from the frozen K0 executor and checked canonical/derived files:

```text
seme-k0-linux-amd64 k0-module-lowerer.k0 k0-module-lowerer.seme B.k0
seme-k0-linux-amd64 B.k0                     k0-module-lowerer.seme C.k0
compare k0-module-lowerer.k0 B.k0
compare B.k0 C.k0

seme-k0-linux-amd64 C.k0 g1-compiler.seme           rebuilt-g1.k0
seme-k0-linux-amd64 C.k0 kernel-wire-validator.seme rebuilt-validator.k0
seme-k0-linux-amd64 C.k0 kernel-migrate-v0.seme     rebuilt-migrator.k0
compare each rebuilt tool with its checked K0 artifact
```

A, B, and C have SHA-256
`dfd62f219bd99f220df853af6f512c23b2b6f0e4f923dc452c5c068574913efe`.

## Continuing development

A readable compiler edit follows this closed path:

```text
seme-k0-linux-amd64 g1-compiler.k0 edited.g1 edited.seme
seme-k0-linux-amd64 kernel-wire-validator.k0 edited.seme
seme-k0-linux-amd64 k0-module-lowerer.k0 edited.seme edited.k0
```

The handoff proof changed a real layout-buffer capacity in the readable lowerer
projection from `1048576` to `1048575`, preserving all stable identities. The
closed path produced a distinct compiler artifact with SHA-256
`f8ce40167383a6872f49eeb762c27d0f03b1a07b134b29084fe5e74d41d5e8bc`.
That edited compiler reproduced itself byte-for-byte and continued to pass the
return, call, and P2 buffer execution fixtures. The edit was a proof only and is
not the checked source.

## Conformance gate

The frozen gate covers all checked fixtures in `conformance/g1`,
`conformance/kernel-v1`, and `conformance/k0-module`. In particular it requires:

- exact A/B/C compiler equality;
- exact rebuilding of the G1 compiler, validator, and migrator by C;
- execution of direct return, arithmetic, locals, backward loops, forward
  branches, multiple functions, calls, P2 buffers, and P2 file copying;
- identical valid artifacts across A/B/C;
- status 65 across A/B/C for invalid opcodes and semantic references;
- G1 rejection of malformed identities, counts, and canonical ordering;
- Kernel rejection of malformed integers, identities, ordering, tags,
  references, reserved meta-schema relationships, and trailing bytes;
- exact unknown-field preservation and v0-to-v1 migration behavior.

Frozen Core 1 specifies deterministic bootstrap status diagnostics: `64` invalid
invocation, `65` malformed/invalid program, and `74` operating-system I/O
failure. Structured diagnostics were subsequently completed by the canonical
located validator and frozen with Kernel v1 without reopening S1 or assembly.

## Scope

“No more assembly” here means no assembly changes are required to continue Seme
compiler development on Linux amd64. A new unsupported platform still needs a
conforming executor emitted by Seme or its own independently auditable ignition
seed. Self-hosting does not by itself prove universal language, package,
mechanic, or platform support.
