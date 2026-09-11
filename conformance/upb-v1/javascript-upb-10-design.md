# JavaScript UPB-10 acceptance design

Status: implemented and claimed by `scripts/check-javascript-upb-10.sh` on
2026-09-11.

JavaScript UPB-10 extends the cumulative native ES-module project through the
same neutral Target Contract v1 and Project Contract v13 used by other source
providers. Target planning consumes authenticated Project-v12 canonical
authority, never JavaScript filenames or syntax.

The bounded project yields 35 requirements. Thirty-one canonical constructs
have exact Wasm/Pulp realizations. DurablePort, TransportPort, live clock
sampling, and external effect delivery remain four typed JavaScript-host native
islands. Exact-only policy therefore produces 31 exact and four impossible
resolutions, zero boundaries, and no deployment.

Realization rule identities carry the explicit `javascript-upb10` namespace.
This records which provider supplied target evidence without adding JavaScript
to Core, Target v1, Project v13, or the canonical application graph. The legacy
Go namespace remains the default only to preserve existing authenticated Go
artifacts.

The cumulative claim requires all seven UPB evidence classes. It reruns the
complete mapped JavaScript UPB-09 gate, independently reproduces Target v1 and
Project v13, validates placement/deployment packages, and builds two identical
source-free deployments. It also proves exact-only non-publication, plan-tamper
rejection, existing-destination rejection, and closed canonical envelopes.

This cell does not claim automatic native-island execution, arbitrary
JavaScript projects, broad npm compatibility, ambient WASI capabilities, or a
general JavaScript runtime. It proves honest target placement for the bounded
cumulative project.
