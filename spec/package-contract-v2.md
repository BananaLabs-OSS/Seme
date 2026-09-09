# Package Contract v2

Package Contract v2 is an additive child of Package Contract v1. It preserves
every v1 schema and field identity and adds a source-aware detail graph for
project-scale package fidelity. Package v1 remains the authority for callable
interfaces, semantic dependencies, effects, runtime assumptions, and fidelity
mappings.

The v2 detail layer distinguishes complete declaration membership from the
public interface. A declaration is exported only when
`declaration_member.export_name` is present. Visibility codes are neutral:
`0` package, `1` project, and `2` public. Import-class codes are `0` local and
`1` external. Adapters must reject or explicitly refine language-specific
visibility that these codes cannot represent.

| Schema | ID | Fields |
|---|---|---|
| PackageGraph | `b020` | packages `b200`, content revision `b201` |
| PackageDetail | `b021` | v1 package `b210`, members `b211`, imports `b212`, origins `b213`, revision `b214` |
| DeclarationMember | `b022` | declaration `b220`, name `b221`, visibility `b222`, optional export name `b223`, origin `b224` |
| Visibility | `b023` | code `b230` |
| ImportBinding | `b024` | optional alias `b240`, requested identity `b241`, class `b242`, optional local package `b243`, optional external dependency `b244`, origin `b245` |
| ImportClass | `b025` | code `b250` |
| SourceOrigin | `b026` | source-unit identity `b260`, path `b261`, SHA-256 digest `b262`, byte range `b263-b264`, line/column range `b265-b268` |

`source_unit_identity` deliberately has no Project schema constraint. This
avoids a Package/Project contract cycle. A composed future Project contract
must resolve it against its source inventory and require matching normalized
path and digest. Package-only validation can validate shape and range but
cannot claim that a Project source unit exists.

Package v2 does not alter v1 package revisions. Detail and graph revisions must
separately commit to their canonical closures when instance support is added.
No current Project contract is changed by this declaration-only milestone.
