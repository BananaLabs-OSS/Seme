# Go UAB-02 gap audit

Go UAB-02 is not certified. Core v30 contains neutral canonical schemas for
signed i64, Boolean, text, byte sequences, records, explicit results, fixed
arrays, slices, and runtime-keyed maps. Existing Core gates provide substantial
isolated Go lift, native, JavaScript projection, Wasm, and Pulp evidence for
most of that vocabulary.

The full frozen cell nevertheless has two architectural gaps:

1. Core v30 has no distinct option type or some/none constructors. Encoding an
   option as a nullable Go pointer, `(T, bool)`, a sentinel, or `Result<T, E>`
   would erase meaning and is not acceptable ecosystem fidelity.
2. The compositional Go session provider and direct Go projector do not yet
   support the existing `BytesType`, `ResultType`, `ResultOk`, or `ResultError`
   schemas. Those schemas currently participate in the older bounded
   application/target path, not the general source bridge used by UAB-v1.

The session provider now regression-tests that `[]byte` and a Go `(T, bool)`
option-shaped return reject with located diagnostics. This prevents either gap
from silently receiving string, slice-of-i64, or multiple-return semantics.

Records, arrays, slices, and maps still require direct canonical-to-Go
projection and combined five-evidence gates after the neutral option semantics
are introduced. Passing their isolated Core gates cannot compensate for a
missing mandatory member of UAB-02.
