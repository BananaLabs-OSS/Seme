// Package orderedtransportplacement verifies the deliberately narrow Pulp
// placement available to an Ordered Transport semantic planner. It does not
// model or certify an authoritative TransportPort.
package orderedtransportplacement

import (
	"fmt"
	"sort"
	"strings"
)

const (
	PinnedPulpCommit     = "acc66ca61fe69c5f2c4093bc55e13aeac6dcc001"
	PinnedManifestSHA256 = "e88c5c9fb7d4c839090dcbc546d838466d6a68b8d818ce3fa9e8da7efeea826c"
	PlannerProvider      = "seme.function-composite-v1"
	PlannerCarrier       = "pulp_on_call"
)

// Candidate is authenticated deployment evidence, not a request to infer a
// mapping from similarly named runtime mechanisms.
type Candidate struct {
	PulpCommit     string
	ManifestSHA256 string
	Provider       string
	Carrier        string
	Capabilities   []string
	Extensions     []string
	Synchronous    bool
	OpaqueBytes    bool
}

// VerifyPlanner accepts only the pinned, capability-free synchronous opaque
// call carrier. Callers must authenticate the commit and manifest bytes before
// constructing Candidate.
func VerifyPlanner(candidate Candidate) error {
	if candidate.PulpCommit != PinnedPulpCommit {
		return fmt.Errorf("ordered_transport.pulp_commit: got %q", candidate.PulpCommit)
	}
	if candidate.ManifestSHA256 != PinnedManifestSHA256 {
		return fmt.Errorf("ordered_transport.manifest_digest: got %q", candidate.ManifestSHA256)
	}
	if candidate.Provider != PlannerProvider {
		return fmt.Errorf("ordered_transport.provider: got %q", candidate.Provider)
	}
	if candidate.Carrier != PlannerCarrier {
		return fmt.Errorf("ordered_transport.carrier: %q is not the synchronous opaque planner carrier", candidate.Carrier)
	}
	if !candidate.Synchronous || !candidate.OpaqueBytes {
		return fmt.Errorf("ordered_transport.fidelity: planner carrier must be synchronous and opaque")
	}
	if len(candidate.Capabilities) != 0 {
		return fmt.Errorf("ordered_transport.capabilities: pure planner must grant none: %s", joined(candidate.Capabilities))
	}
	if len(candidate.Extensions) != 0 {
		return fmt.Errorf("ordered_transport.extensions: pure planner must pin no extensions: %s", joined(candidate.Extensions))
	}
	return nil
}

// VerifyAuthoritativeTransportPort fails closed. The pinned Pulp revision has
// no authenticated provider for Ordered Transport v1 whole-frame receive/send
// acceptance or send-retry evidence. A planner placement cannot be promoted by
// analogy to an authoritative port.
func VerifyAuthoritativeTransportPort(candidate Candidate) error {
	if err := VerifyPlanner(candidate); err != nil {
		return err
	}
	return fmt.Errorf("ordered_transport.transport_port_unsupported: pinned Pulp supplies only %s; no framed receive/send provider", PlannerCarrier)
}

func joined(values []string) string {
	copyOf := append([]string(nil), values...)
	sort.Strings(copyOf)
	return strings.Join(copyOf, ",")
}
