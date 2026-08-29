# K0 semantic lifter

This construction tool converts a checked K0-P2 image into the canonical K0
semantic module's G1 projection. It derives function, instruction, branch, and
call identities while discarding positional branch witnesses and call indices.

It is not authoritative and is not part of the frozen execution path. Its
conformance requirement is exact canonical reproduction: lifting the frozen
Kernel validator and compiling the result must reproduce the checked canonical
`.seme` graph byte-for-byte.
