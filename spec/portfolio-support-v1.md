# Portfolio Support v1

Seme distinguishes project accounting from full semantic lifting. A project is
not silently excluded merely because its language provider or target lowering
is incomplete.

`./seme audit ROOT` inventories each immediate project and records:

- detected language ecosystems and build files;
- the provider currently available or required;
- fidelity (`native` for the initial compatibility island);
- execution strategy;
- whether semantic-provider work remains.

The first audit of `/home/sirniklas/Projects` discovered 100 non-hidden project
directories: 80 containing Go modules, 12 JavaScript/TypeScript projects, five
Rust projects, and four Java/Kotlin projects. Some projects contain multiple
ecosystems, so language counts overlap. Seventy-one are Go-only provider
candidates, sixteen need an additional or new provider, and thirteen currently
have metadata-only accounting.

These numbers describe discovered build markers, not successful full lifts.
The Go provider still needs broader dependency/build-constraint handling, and
the other ecosystems need provider contracts. Native islands are the immediate
execution fallback while that work proceeds.
