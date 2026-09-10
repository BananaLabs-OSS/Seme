package orderedtransportplacement

import (
	"strings"
	"testing"
)

func exactPlanner() Candidate {
	return Candidate{PulpCommit: PinnedPulpCommit, ManifestSHA256: PinnedManifestSHA256, Provider: PlannerProvider, Carrier: PlannerCarrier, Synchronous: true, OpaqueBytes: true}
}

func TestOnlyExactPurePlannerPlacementIsAccepted(t *testing.T) {
	if err := VerifyPlanner(exactPlanner()); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*Candidate)
	}{
		{"unpinned-commit", func(c *Candidate) { c.PulpCommit = "HEAD" }},
		{"substituted-manifest", func(c *Candidate) { c.ManifestSHA256 = strings.Repeat("0", 64) }},
		{"provider-alias", func(c *Candidate) { c.Provider = "seme.transport.v1" }},
		{"step-event", func(c *Candidate) { c.Carrier = "pulp_step/StepEvent" }},
		{"http", func(c *Candidate) { c.Carrier = "http" }},
		{"websocket", func(c *Candidate) { c.Carrier = "websocket" }},
		{"stdin", func(c *Candidate) { c.Carrier = "stdin" }},
		{"stdout", func(c *Candidate) { c.Carrier = "stdout" }},
		{"raw-call", func(c *Candidate) { c.Carrier = "pulp.call_raw" }},
		{"asynchronous", func(c *Candidate) { c.Synchronous = false }},
		{"decoded", func(c *Candidate) { c.OpaqueBytes = false }},
		{"fs", func(c *Candidate) { c.Capabilities = []string{"storage.fs"} }},
		{"sqlite", func(c *Candidate) { c.Capabilities = []string{"storage.sqlite"} }},
		{"unpinned-extension", func(c *Candidate) { c.Extensions = []string{"example.invalid/transport"} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := exactPlanner()
			test.mutate(&candidate)
			if err := VerifyPlanner(candidate); err == nil {
				t.Fatal("invalid placement accepted")
			}
		})
	}
}

func TestPlannerIsNeverAnAuthoritativeTransportPort(t *testing.T) {
	err := VerifyAuthoritativeTransportPort(exactPlanner())
	if err == nil || !strings.Contains(err.Error(), "transport_port_unsupported") {
		t.Fatalf("got %v", err)
	}
}
