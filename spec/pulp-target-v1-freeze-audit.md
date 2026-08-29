# Pulp target v1 freeze audit

| Property | Result | Evidence |
|---|---|---|
| Real runtime | pass | Pulp's normal manifest loader, host, wazero runtime, lifecycle, allocator, and provider-call path execute the pinned artifact. |
| Request/response | pass | One cell instance processes three distinct Application Wire v1 records and returns the expected Boolean responses, including modular overflow. |
| Effect delivery | pass | Granted `observability.log` produces exactly one decision event per provider call. |
| Capability enforcement | pass | The identical artifact without a grant receives status 99 and traps during the call without an event. |
| Lifecycle compatibility | pass | Generated memory, provider export, and four required lifecycle exports satisfy Pulp's cell contract and shutdown is clean. |
| Plan continuity | pass | The artifact remains reproduced from the checked adapted Seme target plan. |

Pulp runtime evidence is pinned at commit `f4d15bb`; the conformance gate
requires that commit in the selected Pulp repository.

The proof uses a dedicated conformance deployment binary in Pulp and a fixed
versioned request record. Schema-derived codecs and production capability
providers remain future profiles.
