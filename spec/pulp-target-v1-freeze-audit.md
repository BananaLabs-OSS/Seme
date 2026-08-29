# Pulp target v1 freeze audit

| Property | Result | Evidence |
|---|---|---|
| Real runtime | pass | Pulp's normal manifest loader, host, wazero runtime, lifecycle, and step loop execute the pinned artifact. |
| Effect delivery | pass | Granted `observability.log` produces exactly one decision event during initialization. |
| Capability enforcement | pass | The identical artifact without a grant receives status 99, traps, and fails initialization without an event. |
| Lifecycle compatibility | pass | Generated memory and four required exports satisfy Pulp's cell contract and shutdown is clean. |
| Plan continuity | pass | The artifact remains reproduced from the checked adapted Seme target plan. |

Pulp runtime evidence is pinned at commit `97b1c8c`; the conformance gate
requires that commit in the selected Pulp repository.

The proof uses a dedicated conformance deployment binary in Pulp and a fixed
initialization vector. General calls and production capability providers remain
future profiles.
