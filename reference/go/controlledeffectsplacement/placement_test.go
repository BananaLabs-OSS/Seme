package controlledeffectsplacement

import (
	"reflect"
	"seme.local/reference/controlledeffectsinstance"
	"testing"
)

func model() controlledeffectsinstance.Model {
	return controlledeffectsinstance.Model{Clock: controlledeffectsinstance.Clock{Identity: "clock.injected.unix-milliseconds.v1"}, Random: controlledeffectsinstance.Random{Identity: "random.seeded.lcg-48271-plus-1.v1"}, ExternalEffect: controlledeffectsinstance.ExternalBooleanEffect{Identity: "observability.log", CapabilityIdentity: "observability.log"}}
}
func TestExactSplitPlacement(t *testing.T) {
	m := model()
	if e := verifyHost(m, ExactHost(m)); e != nil {
		t.Fatal(e)
	}
	for _, target := range []string{"canonical", "wasm", "pulp"} {
		if e := VerifyPure(ExactPure(target)); e != nil {
			t.Fatal(target, e)
		}
	}
	if e := VerifyObservedParity(ExactObservedParity()); e != nil {
		t.Fatal(e)
	}
}
func TestObservedHarnessIsNotExternalDelivery(t *testing.T) {
	tests := []func(*ObservedCandidate){func(c *ObservedCandidate) { c.ManifestSHA256 = PinnedManifestSHA256 }, func(c *ObservedCandidate) { c.RunnerSHA256 = "00" }, func(c *ObservedCandidate) { c.Provider = PureProvider }, func(c *ObservedCandidate) { c.Capability = "entropy.read" }, func(c *ObservedCandidate) { c.CapabilityProvider = "pulp.observability" }, func(c *ObservedCandidate) { c.Fidelity = "exact-declared-provider" }, func(c *ObservedCandidate) { c.ObservationMeaning = "external-delivery" }, func(c *ObservedCandidate) { c.Synchronous = false }}
	for i, edit := range tests {
		c := ExactObservedParity()
		edit(&c)
		if VerifyObservedParity(c) == nil {
			t.Fatalf("adversary %d", i)
		}
	}
}
func TestHostRejectsMissingExtraAndAmbientAuthority(t *testing.T) {
	m := model()
	tests := []func(*HostCandidate){func(c *HostCandidate) { c.Capabilities = nil }, func(c *HostCandidate) { c.Capabilities = append(c.Capabilities, "entropy.read") }, func(c *HostCandidate) { c.AmbientProviders = []string{"wasi.clock"} }, func(c *HostCandidate) { c.Provider = "wasi.preview1.clock_time_get" }, func(c *HostCandidate) { c.Fidelity = "exact" }, func(c *HostCandidate) { c.InjectedClock = false }, func(c *HostCandidate) { c.ExternalEffectDelivery = false }}
	for i, edit := range tests {
		c := ExactHost(m)
		edit(&c)
		if e := verifyHost(m, c); e == nil {
			t.Fatalf("adversary %d", i)
		}
	}
}
func TestPureRejectsSubstitutionAndFalseExactness(t *testing.T) {
	tests := []func(*PureCandidate){func(c *PureCandidate) { c.AmbientProviders = []string{"wasi.clock"} }, func(c *PureCandidate) { c.AmbientProviders = []string{"entropy.read"} }, func(c *PureCandidate) { c.Capabilities = []string{"observability.log"} }, func(c *PureCandidate) { c.Extensions = []string{"unrelated/pulp-provider"} }, func(c *PureCandidate) { c.Provider = "pulp.http" }, func(c *PureCandidate) { c.Carrier = "pulp.call_raw" }, func(c *PureCandidate) { c.Fidelity = "exact" }, func(c *PureCandidate) { c.Roles = []string{"planner"} }, func(c *PureCandidate) { c.Roles = []string{"planner", "replay", "replay"} }, func(c *PureCandidate) { c.PulpCommit = "HEAD" }, func(c *PureCandidate) { c.ManifestSHA256 = "00" }, func(c *PureCandidate) { c.Pure = false }}
	for i, edit := range tests {
		c := ExactPure("pulp")
		edit(&c)
		if e := VerifyPure(c); e == nil {
			t.Fatalf("adversary %d", i)
		}
	}
	if VerifyPure(PureCandidate{Target: "wasi", Provider: PureProvider}) == nil {
		t.Fatal("ambient target accepted")
	}
}
func TestZeroInstanceAndBuildAreAtomic(t *testing.T) {
	if VerifyHost(controlledeffectsinstance.Inputs{}, HostCandidate{}) == nil {
		t.Fatal("zero instance")
	}
	got, e := Build(controlledeffectsinstance.Inputs{}, HostCandidate{}, PureCandidate{}, PureCandidate{}, PureCandidate{}, ObservedCandidate{})
	if e == nil || !reflect.DeepEqual(got, Evidence{}) {
		t.Fatal("partial evidence")
	}
}
