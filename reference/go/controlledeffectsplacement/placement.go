// Package controlledeffectsplacement authenticates the deliberately split
// placement of Controlled Effects v1. Live clock sampling and effect delivery
// remain a Go host boundary; planner and replay semantics remain pure.
package controlledeffectsplacement

import (
	"fmt"
	"sort"
	"strings"

	"seme.local/reference/controlledeffectsinstance"
)

const (
	PinnedPulpCommit     = "acc66ca61fe69c5f2c4093bc55e13aeac6dcc001"
	PinnedManifestSHA256 = "e88c5c9fb7d4c839090dcbc546d838466d6a68b8d818ce3fa9e8da7efeea826c"
	PureProvider         = "seme.function-composite-v1"
)

type HostCandidate struct {
	Provider, Placement, Fidelity                       string
	Capabilities, AmbientProviders                      []string
	InjectedClock, ExplicitSeed, ExternalEffectDelivery bool
}
type PureCandidate struct {
	Target, Provider, Carrier, Fidelity, PulpCommit, ManifestSHA256 string
	Roles, Capabilities, AmbientProviders, Extensions               []string
	Pure, Synchronous, OpaqueBytes                                  bool
}
type Evidence struct {
	ClockIdentity, RandomIdentity, EffectIdentity string
	Host                                          HostCandidate
	Canonical, Wasm, Pulp                         PureCandidate
}

func VerifyHost(in controlledeffectsinstance.Inputs, c HostCandidate) error {
	if err := controlledeffectsinstance.Validate(in); err != nil {
		return fmt.Errorf("controlled_effects_placement.instance:%w", err)
	}
	return verifyHost(in.Model, c)
}
func verifyHost(m controlledeffectsinstance.Model, c HostCandidate) error {
	if c.Provider != "go.controlled-effects-host-v1" || c.Placement != "go-host-boundary" || c.Fidelity != "exact-declared-provider" || !c.InjectedClock || !c.ExplicitSeed || !c.ExternalEffectDelivery {
		return fmt.Errorf("controlled_effects_placement.host")
	}
	if !sameSet(c.Capabilities, []string{m.Clock.Identity, m.ExternalEffect.CapabilityIdentity}) || len(c.AmbientProviders) != 0 {
		return fmt.Errorf("controlled_effects_placement.host_capabilities:%s", joined(c.Capabilities))
	}
	return nil
}
func VerifyPure(c PureCandidate) error {
	carrier := ""
	opaque := true
	switch c.Target {
	case "canonical":
		carrier, opaque = "canonical-value-call", false
	case "wasm":
		carrier = "wasm-export"
	case "pulp":
		carrier = "pulp_on_call"
	default:
		return fmt.Errorf("controlled_effects_placement.target")
	}
	if c.Provider != PureProvider || c.Carrier != carrier || c.Fidelity != "exact-bounded-semantics" || !c.Pure || !c.Synchronous || c.OpaqueBytes != opaque || !sameSet(c.Roles, []string{"planner", "replay"}) || len(c.Capabilities) != 0 || len(c.AmbientProviders) != 0 || len(c.Extensions) != 0 {
		return fmt.Errorf("controlled_effects_placement.pure")
	}
	if c.Target == "pulp" {
		if c.PulpCommit != PinnedPulpCommit || c.ManifestSHA256 != PinnedManifestSHA256 {
			return fmt.Errorf("controlled_effects_placement.pulp_pin")
		}
	} else if c.PulpCommit != "" || c.ManifestSHA256 != "" {
		return fmt.Errorf("controlled_effects_placement.false_pin")
	}
	return nil
}
func Build(in controlledeffectsinstance.Inputs, h HostCandidate, canonical, wasm, pulp PureCandidate) (Evidence, error) {
	if err := VerifyHost(in, h); err != nil {
		return Evidence{}, err
	}
	for _, c := range []PureCandidate{canonical, wasm, pulp} {
		if err := VerifyPure(c); err != nil {
			return Evidence{}, err
		}
	}
	m := in.Model
	return Evidence{ClockIdentity: m.Clock.Identity, RandomIdentity: m.Random.Identity, EffectIdentity: m.ExternalEffect.Identity, Host: h, Canonical: canonical, Wasm: wasm, Pulp: pulp}, nil
}
func ExactHost(m controlledeffectsinstance.Model) HostCandidate {
	return HostCandidate{Provider: "go.controlled-effects-host-v1", Placement: "go-host-boundary", Fidelity: "exact-declared-provider", Capabilities: []string{m.Clock.Identity, m.ExternalEffect.CapabilityIdentity}, InjectedClock: true, ExplicitSeed: true, ExternalEffectDelivery: true}
}
func ExactPure(target string) PureCandidate {
	c := PureCandidate{Target: target, Provider: PureProvider, Fidelity: "exact-bounded-semantics", Roles: []string{"planner", "replay"}, Pure: true, Synchronous: true, OpaqueBytes: true}
	switch target {
	case "canonical":
		c.Carrier = "canonical-value-call"
		c.OpaqueBytes = false
	case "wasm":
		c.Carrier = "wasm-export"
	case "pulp":
		c.Carrier = "pulp_on_call"
		c.PulpCommit = PinnedPulpCommit
		c.ManifestSHA256 = PinnedManifestSHA256
	}
	return c
}
func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x, y := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] == "" || x[i] != y[i] || (i > 0 && x[i] == x[i-1]) {
			return false
		}
	}
	return true
}
func joined(x []string) string {
	y := append([]string(nil), x...)
	sort.Strings(y)
	return strings.Join(y, ",")
}
