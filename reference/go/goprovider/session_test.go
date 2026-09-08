package goprovider

import (
	"os"
	"strings"
	"testing"
)

func TestIncrementalSessionRetainsLastValidGraph(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v12/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	valid := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/session", Files: map[string]string{
		"value.go": "package value\nfunc Combine(left, right int64) int64 { return (left + 1) * right }\n",
	}})
	if !valid.Accepted || !valid.Valid || valid.Disposition != "accepted-valid" || valid.LastValidRevision != 1 || len(valid.Sources) != 1 || len(valid.ContentDigest) != 64 {
		t.Fatalf("valid result = %#v", valid)
	}
	if valid.Sources[0].Name != "Combine" || valid.Sources[0].Document != "value.go" || valid.Sources[0].Line != 2 || valid.Sources[0].ID == "" {
		t.Fatalf("source mapping = %#v", valid.Sources[0])
	}
	baseline := valid.CanonicalG1

	incomplete := session.Apply(DocumentSnapshot{Revision: 2, PackagePath: "example.test/session", Files: map[string]string{
		"value.go": "package value\nfunc Combine(left, right int64) int64 { return (left +\n",
	}})
	if !incomplete.Accepted || incomplete.Valid || incomplete.Disposition != "accepted-invalid" || incomplete.LastValidRevision != 1 || incomplete.CanonicalG1 != baseline {
		t.Fatalf("incomplete result did not retain baseline: %#v", incomplete)
	}
	if len(incomplete.Diagnostics) == 0 || incomplete.Diagnostics[0].Code != "go.parse" || incomplete.Diagnostics[0].Line == 0 {
		t.Fatalf("parse diagnostics = %#v", incomplete.Diagnostics)
	}

	typeInvalid := session.Apply(DocumentSnapshot{Revision: 3, PackagePath: "example.test/session", Files: map[string]string{
		"value.go": "package value\nfunc Combine(left, right int64) int64 { return left + missing }\n",
	}})
	if typeInvalid.Valid || typeInvalid.LastValidRevision != 1 || typeInvalid.CanonicalG1 != baseline || len(typeInvalid.Diagnostics) == 0 || typeInvalid.Diagnostics[0].Code != "go.type" {
		t.Fatalf("type-invalid result = %#v", typeInvalid)
	}

	recovered := session.Apply(DocumentSnapshot{Revision: 4, PackagePath: "example.test/session", Files: map[string]string{
		"value.go": "package value\nfunc Combine(left, right int64) int64 { return (left - 1) * right }\n",
	}})
	if !recovered.Valid || recovered.LastValidRevision != 4 || recovered.CanonicalG1 == baseline || recovered.Sources[0].ID != valid.Sources[0].ID {
		t.Fatalf("recovered result = %#v", recovered)
	}

	stale := session.Apply(DocumentSnapshot{Revision: 4, PackagePath: "example.test/session", Files: map[string]string{
		"value.go": "package value\nfunc Combine(left, right int64) int64 { return left }\n",
	}})
	if stale.Accepted || stale.Disposition != "rejected-stale" || stale.CanonicalG1 != recovered.CanonicalG1 || stale.LastValidRevision != 4 || stale.Diagnostics[0].Code != "session.stale_revision" {
		t.Fatalf("stale result = %#v", stale)
	}
}

func TestIncrementalSessionLiftsSupportedDeclarationsAndReportsOpaqueOnes(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v12/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := DocumentSnapshot{Revision: 7, PackagePath: "example.test/multi", Files: map[string]string{
		"first.go":  "package multi\nfunc Enabled(value int64) bool { return true && value <= 9 }\n",
		"second.go": "package multi\nfunc Unsupported(value string) string { return value }\n",
	}}
	first := session.Apply(snapshot)
	if !first.Valid || len(first.Sources) != 1 || first.Sources[0].Name != "Enabled" || len(first.Diagnostics) != 1 || first.Diagnostics[0].Severity != "warning" {
		t.Fatalf("partial supported result = %#v", first)
	}
	if strings.Contains(first.CanonicalG1, "Unsupported") {
		t.Fatal("unsupported declaration leaked into canonical graph")
	}
	other, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	second := other.Apply(snapshot)
	if second.CanonicalG1 != first.CanonicalG1 || second.ContentDigest != first.ContentDigest {
		t.Fatal("equal snapshots did not produce deterministic state")
	}
}

func TestIncrementalSessionLiftsTotalReturnControlAndRetainsIt(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v13/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	valid := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/control", Files: map[string]string{
		"control.go": "package control\nfunc Choose(enabled bool, value int64) int64 {\nif enabled { return value + 1 }\nreturn value - 1\n}\n",
	}})
	if !valid.Valid || valid.LastValidRevision != 1 || len(valid.Sources) != 1 || valid.Sources[0].Name != "Choose" {
		t.Fatalf("control result = %#v", valid)
	}
	if !strings.Contains(valid.CanonicalG1, "000000000000000000000000000090c0") {
		t.Fatal("canonical control node missing")
	}
	baseline := valid.CanonicalG1
	invalid := session.Apply(DocumentSnapshot{Revision: 2, PackagePath: "example.test/control", Files: map[string]string{
		"control.go": "package control\nfunc Choose(enabled bool, value int64) int64 {\nif enabled { return value + 1 }\n}\n",
	}})
	if !invalid.Accepted || invalid.Valid || invalid.LastValidRevision != 1 || invalid.CanonicalG1 != baseline || len(invalid.Sources) != 1 {
		t.Fatalf("invalid control edit = %#v", invalid)
	}
	if len(invalid.Diagnostics) == 0 || invalid.Diagnostics[0].Code != "go.type" || invalid.Diagnostics[0].Line == 0 {
		t.Fatalf("control diagnostic = %#v", invalid.Diagnostics)
	}
}
