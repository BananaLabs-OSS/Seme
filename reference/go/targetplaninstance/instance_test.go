package targetplaninstance

import (
	"bytes"
	"os"
	"os/exec"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/wire"
)

func TestEmittedPlanPassesIndependentFoundationValidator(t *testing.T) {
	for _, allowed := range [][]Fidelity{{Exact, Adapted}, {Exact}} {
		input := fixture(t)
		input.Model.Allowed = allowed
		artifact, err := Emit(input)
		if err != nil {
			t.Fatal(err)
		}
		path := t.TempDir() + "/plan.seme"
		if err = os.WriteFile(path, artifact, 0o600); err != nil {
			t.Fatal(err)
		}
		command := exec.Command("../../../bootstrap/seme-k0-linux-amd64", "../../../modules/foundation/v1/validator.k0", path)
		if output, runErr := command.CombinedOutput(); runErr != nil {
			t.Fatalf("foundation validator: %v\n%s", runErr, output)
		}
	}
}

func TestDeterministicResolutionPreservesFidelityAndBoundaries(t *testing.T) {
	in := fixture(t)
	first, err := Emit(in)
	if err != nil {
		t.Fatal(err)
	}
	permuted := in
	permuted.Model.Requirements = reverse(permuted.Model.Requirements)
	permuted.Model.Target.Rules = reverse(permuted.Model.Target.Rules)
	permuted.Model.Allowed = reverse(permuted.Model.Allowed)
	permuted.Model.Boundaries = reverse(permuted.Model.Boundaries)
	second, err := Emit(permuted)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("nondeterministic plan", err)
	}
	in.Artifact = first
	if err = Validate(in); err != nil {
		t.Fatal(err)
	}
	graph, err := wire.Decode(first)
	if err != nil {
		t.Fatal(err)
	}
	resolutions := bySchema(graph, id("c013"))
	if len(resolutions) != 2 || len(bySchema(graph, id("c015"))) != 1 || len(bySchema(graph, id("c014"))) != 1 {
		t.Fatal("plan cardinality")
	}
	seen := map[uint64]int{}
	for _, resolution := range resolutions {
		seen[resolution.Fields[id("c133")].Unsigned]++
	}
	if seen[uint64(Exact)] != 1 || seen[uint64(Adapted)] != 1 {
		t.Fatal("fidelity changed", seen)
	}
	plan := bySchema(graph, id("c014"))[0]
	if plan.Fields[id("c144")].Tag != 2 {
		t.Fatal("executable mixed plan rejected")
	}
}

func TestForbiddenAndMalformedPlansRejectAtomically(t *testing.T) {
	in := fixture(t)
	in.Model.Allowed = []Fidelity{Exact}
	artifact, err := Emit(in)
	if err != nil {
		t.Fatal(err)
	}
	graph, _ := wire.Decode(artifact)
	plan := bySchema(graph, id("c014"))[0]
	if plan.Fields[id("c144")].Tag != 1 {
		t.Fatal("impossible plan marked executable")
	}
	seenImpossible := false
	for _, resolution := range bySchema(graph, id("c013")) {
		if resolution.Fields[id("c133")].Unsigned == uint64(Impossible) {
			seenImpossible = true
			diagnostics := resolution.Fields[id("c135")].List
			if len(diagnostics) != 1 {
				t.Fatal("impossible resolution lacks one diagnostic")
			}
			diagnostic := graph.Entities[diagnostics[0].Reference]
			if diagnostic.Schema != id("1a") || diagnostic.Fields[id("1a5")].Unsigned != 2 {
				t.Fatal("invalid impossible diagnostic")
			}
			rule := graph.Entities[diagnostic.Fields[id("1a0")].Reference]
			if rule.Schema != id("19") || string(rule.Fields[id("190")].Bytes) != "target.no_permitted_realization" {
				t.Fatal("invalid impossible diagnostic rule")
			}
		}
	}
	if !seenImpossible {
		t.Fatal("forbidden adapted rule was relabeled")
	}

	cases := []func(*Inputs){
		func(x *Inputs) { x.Model.Requirements[0].Construct = id("ffff") },
		func(x *Inputs) { x.Model.Requirements = append(x.Model.Requirements, x.Model.Requirements[0]) },
		func(x *Inputs) { x.Model.Target.Rules[1].Fidelity = Impossible },
		func(x *Inputs) { x.Model.Boundaries[0].Consumer = id("ffff") },
		func(x *Inputs) { x.Model.Boundaries[0].Requirement = "exact" },
		func(x *Inputs) { x.Model.Allowed = append(x.Model.Allowed, Exact) },
	}
	for index, mutate := range cases {
		bad := fixture(t)
		mutate(&bad)
		if partial, emitErr := Emit(bad); emitErr == nil || partial != nil {
			t.Fatalf("case %d accepted or returned partial artifact", index)
		}
	}
	badArtifact := fixture(t)
	badArtifact.Artifact = append([]byte(nil), artifact...)
	badArtifact.Artifact[len(badArtifact.Artifact)-1] ^= 1
	if err = Validate(badArtifact); err == nil {
		t.Fatal("tampered artifact accepted")
	}
}

func fixture(t *testing.T) Inputs {
	t.Helper()
	contractBytes, err := os.ReadFile("../../../modules/target/v1/module.seme")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := contractcatalog.ResolveTargetContract(contractBytes)
	if err != nil {
		t.Fatal(err)
	}
	authority, err := os.ReadFile("../../../modules/controlled-effects/v1/module.seme")
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]wire.ID{"root": id("13100"), "exact": id("13101"), "adapted": id("13102"), "property": id("13103"), "provider": id("13104"), "interface": id("13105"), "transport": id("13106"), "evidence": id("13107")}
	provider := ids["provider"]
	return Inputs{TargetContract: contract, Authority: authority, Model: Model{
		Root: ids["root"], Target: Target{Name: "test-target", Revision: 1, Rules: []Rule{
			{Identity: "adapt", Construct: ids["adapted"], MaximumRevision: 1, PreservedProperties: []wire.ID{ids["property"]}, Fidelity: Adapted, Dependency: &provider, Evidence: []wire.ID{ids["evidence"]}},
			{Identity: "exact", Construct: ids["exact"], MaximumRevision: 1, PreservedProperties: []wire.ID{ids["property"]}, Fidelity: Exact, Evidence: []wire.ID{ids["evidence"]}},
		}},
		Requirements: []Requirement{
			{Identity: "adapted", Construct: ids["adapted"], MinimumRevision: 1, Properties: []wire.ID{ids["property"]}},
			{Identity: "exact", Construct: ids["exact"], MinimumRevision: 1, Properties: []wire.ID{ids["property"]}},
		},
		Allowed: []Fidelity{Adapted, Exact}, Boundaries: []Boundary{{Requirement: "adapted", Provider: ids["provider"], Consumer: ids["exact"], Interface: ids["interface"], Transport: ids["transport"]}},
	}}
}

func bySchema(graph wire.Envelope, schema wire.ID) []wire.Entity {
	var out []wire.Entity
	for _, entity := range graph.Entities {
		if entity.Schema == schema {
			out = append(out, entity)
		}
	}
	return out
}

func reverse[T any](in []T) []T {
	out := append([]T(nil), in...)
	for left, right := 0, len(out)-1; left < right; left, right = left+1, right-1 {
		out[left], out[right] = out[right], out[left]
	}
	return out
}
