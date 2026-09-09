# Project Contract v2

Project Contract v2 extends v1 with a language-neutral, non-executable source
inventory vocabulary. Its module remains `...e000`; revision `...e002` has
the v1 revision `...e001` as its sole parent.

The inventory records explicit language/toolchain/profile identities,
normalized relative paths, byte sizes, closed classification and preservation
modes, and digest-addressed source units. It
binds separately to a semantic ProjectSnapshot and its 32-byte semantic
revision. Raw source bytes belong in an external digest-addressed bundle, not
in canonical semantic entities.

See `spec/project-contract-v2.md` for the normative boundary and classification
values.
