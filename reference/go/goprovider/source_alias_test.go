package goprovider

import (
	"os"
	"reflect"
	"testing"
)

func TestSourceAliasDiscoveryPreservesTypedTargetAndImportProvenance(t *testing.T) {
	execution, err := os.ReadFile("../../../modules/execution/v36/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewIncrementalSession(execution)
	if err != nil {
		t.Fatal(err)
	}
	r := s.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/alias", PackagePath: "example.test/alias/persistence", Entry: "Build", Files: map[string]string{
		"model/model.go":      "package model\ntype Result[T any, E any] struct { Ok bool; Value T; Error E }\n",
		"persistence/plan.go": "package persistence\nimport \"example.test/alias/model\"\ntype Decision struct { Value int64 }\ntype Plan = model.Result[Decision, int64]\nfunc Build(value int64) Plan { return Plan{Ok:true, Value:Decision{Value:value}} }\n",
	}})
	if !r.Valid {
		t.Fatal(r.Diagnostics)
	}
	var aliases []SourceAliasMetadata
	for _, p := range r.Packages {
		if p.Name == "example.test/alias/persistence" {
			aliases = p.Aliases
		}
	}
	if len(aliases) != 1 {
		t.Fatalf("aliases=%+v", aliases)
	}
	a := aliases[0]
	if a.Name != "Plan" || !a.Exported || a.Package != "example.test/alias/persistence" || a.Target == "" || a.Document != "persistence/plan.go" || a.Line != 4 || !reflect.DeepEqual(a.ReferencedImports, []string{"example.test/alias/model"}) {
		t.Fatalf("alias=%+v", a)
	}
	// Returned metadata is caller-owned even across a stale-session response.
	r.Packages[1].Aliases[0].ReferencedImports[0] = "forged"
	stale := s.Apply(DocumentSnapshot{Revision: 1})
	for _, p := range stale.Packages {
		for _, got := range p.Aliases {
			if got.Name == "Plan" && got.ReferencedImports[0] != "example.test/alias/model" {
				t.Fatal("alias metadata aliases caller mutation")
			}
		}
	}
}
