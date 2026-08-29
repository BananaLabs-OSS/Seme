# Canonical SHA-256 v1

`digest.seme` is Seme's authoritative portable SHA-256 program. It accepts an
input file and output path and writes the 32-byte SHA-256 digest using only
frozen K0 integer, buffer, argument, and filesystem operations.

`digest.s1` is generated construction history. The small Go constructor emits
the repetitive 64-round schedule and constants to prevent transcription errors;
it is not trusted at execution time. `digest.g1` and `digest.k0` are checked
projection and lowering artifacts. The canonical graph lowers back to the
checked K0 image byte-for-byte.

The bootstrap implementation accepts inputs whose SHA-256 padding remains
within K0's one-MiB buffer limit. Larger semantic artifacts will use streaming
target implementations under this same digest contract.
