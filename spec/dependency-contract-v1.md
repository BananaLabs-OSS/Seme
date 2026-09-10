# Dependency Contract v1

Module `0000000000000000000000000000f000`, revision
`0000000000000000000000000000f001`, defines the neutral resolved dependency
closure used by UPB-03.

| Schema | ID | Meaning |
|---|---|---|
| DependencyClosure | `f010` | Root requirements, complete resolutions, immutable content revision |
| RootRequirement | `f011` | Requested identity/range, dependency kind, applicable metadata |
| ResolvedDependency | `f012` | Identity, optional ecosystem, selected version, integrity, source, kind, transitive dependencies, metadata |
| Ecosystem | `f013` | Stable ecosystem identity |
| DependencyKind | `f014` | Closed code: `0` local, `1` ecosystem |
| DependencyMetadata | `f015` | Neutral key/value extension metadata |
| DependencyIntegrity | `f016` | Algorithm and digest bytes |
| DependencySource | `f017` | Source kind, stable identity, and metadata |

Root requirements and resolved dependency lists are complete ordered sets.
Every ecosystem dependency has an ecosystem, version, integrity, and source.
Local dependencies retain the same integrity/source fields so local snapshots
are content-addressed rather than inferred from host paths. Adapters own
ecosystem-specific version syntax and resolution algorithms; this contract
records their resolved meaning without adopting their mechanics.
