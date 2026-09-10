package goconfigurationmanifest

import (
	"os"
	"testing"
)

func TestParseStrictRecursiveSelection(t *testing.T) {
	s := []byte(`{"fields":[],"runtime_inputs":[{"identity":"state","type":{"package":"application","name":"State"}}],"initializers":[{"key":"service","callable":{"package":"service","name":"Assemble"},"dependencies":[],"arguments":[{"kind":"record","record":{"type":{"package":"configuration","name":"Input"},"members":[{"name":"Limit","source":{"kind":"resolved-field","field":"limit"}}]}},{"kind":"runtime-input","runtime":"state"}]}]}`)
	x, err := Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(x.Units) != 1 || x.Units[0].Arguments[0].Record.Members[0].Source.Field != "limit" {
		t.Fatalf("wrong selection %#v", x)
	}
}

func TestUPB05FixtureSelectionIsStrictAndPackageQualified(t *testing.T) {
	b, err := os.ReadFile("../../../fixtures/go-upb05-configuration-overlay/configuration-selection.json")
	if err != nil {
		t.Fatal(err)
	}
	x, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(x.Fields) != 3 || len(x.Runtime) != 1 || len(x.Units) != 3 || x.Fields[1].DefaultProvider == nil || x.Units[2].Callable.Name != "Assemble" {
		t.Fatalf("wrong fixture selection %#v", x)
	}
	for _, u := range x.Units {
		if u.Callable.Package == "" {
			t.Fatal("unqualified callable")
		}
	}
}
func TestParseRejectsUnknownTrailingDepthAndBadID(t *testing.T) {
	for _, x := range [][]byte{[]byte(`{"initializers":[],"unknown":1}`), []byte(`{"initializers":[]} trailing`), []byte(`{"fields":[{"key":"x","owner_package":"p","type":{"id":"bad"},"origin":{},"required":true,"resolution":{"kind":"explicit","value":"bad"}}],"initializers":[{"key":"x","callable":{"package":"p","name":"F"},"dependencies":[],"arguments":[]}]}`)} {
		if got, err := Parse(x); err == nil || len(got.Units) != 0 {
			t.Fatal("accepted malformed manifest")
		}
	}
}
