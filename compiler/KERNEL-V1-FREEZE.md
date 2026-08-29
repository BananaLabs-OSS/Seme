# Kernel v1 freeze

Kernel v1 freezes Seme's structural semantic graph and canonical wire boundary.
It does not freeze the versioned semantic-foundation module carried by that
graph. Constraints, effects, capabilities, provenance, refinement, diagnostics,
memory, concurrency, and other meanings can evolve as schemas without changing
the decoder.

## Frozen authorities

| Artifact | SHA-256 |
|---|---|
| located validator G1 projection | `d5fed4ca8e24de51cc74d9b712e34eff0b4539aa509924ffbe9b952d41e04ccb` |
| located validator canonical graph | `8f1274a9893ff5411584889cd47976f5c2f715e43c7f28eab1acf32cf4c8a9b4` |
| located validator derived K0 | `c77e696ab5d304670049f121cfc2279a20104c3838eeed9e1d3c734a12e95cd4` |
| bootstrap meta-schema graph | `6e75ab9db166cf541328173bec1e3ed1a98c09d5b4b59eaabca21b1b8b800238` |
| v0 migration graph | `b1a51b52cf987a00fc7b46a1f2bceaff0205590a3a023fcd286f39b3372aaa00` |
| v0 migration derived K0 | `a1bdf160b9c2e89eefcad687eb0304cb82abe069dcc5b31856813bd8e7a3da4b` |

The `.seme` graphs are canonical. G1 is a checked readable projection and K0 is
a derived execution artifact. The trusted Linux-amd64 K0 executor remains the
Frozen Core 1 artifact with SHA-256
`ce4a6a282d21a7e152f0015e3dd38ffa665727e2af3db8eb18da7310c5603cac`.

## Frozen guarantees

- shortest-form canonical integers, counts, identities, fields, records, lists,
  values, revisions, and envelopes;
- deterministic entity, field, record, and parent ordering;
- whole-envelope validation before successful preservation;
- local forward-reference resolution and dangling-reference rejection;
- reserved bootstrap Module/Schema/Field relationship validation;
- byte-exact preservation of valid canonical files and unknown module data;
- deterministic canonical diagnostic reports with stable structural location;
- stable process statuses for invalid invocation, invalid input, and I/O failure;
- executable v0 envelope migration;
- independent canonical reproduction through Frozen Core 1.

Run the ordinary gate with `./scripts/check-kernel-v1.sh`. Run
`./scripts/check-kernel-v1.sh --reproduce` for the slower exact validator
reproduction.

## Evolution rule

The wire version, magic, value tags, ordering, identity width, and frozen
bootstrap relationships cannot change under the name Kernel v1. A necessary
incompatible structural change requires Kernel v2 plus an explicit migration.

New schemas, fields, constraints, effects, mechanics, diagnostic rules, and
module validators do not require a new Kernel version. They carry their own
stable identities and versions and must preserve unknown data.
