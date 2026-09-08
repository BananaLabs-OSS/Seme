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
		"second.go": "package multi\nfunc Unsupported(value float64) float64 { return value }\n",
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

func TestIncrementalSessionLiftsStringParametersAndResult(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v14/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/text", Files: map[string]string{
		"text.go": "package text\nfunc Join(left, right string) string { return left + \"λ\" + right }\n",
	}})
	if !result.Valid || result.LastValidRevision != 1 || len(result.Sources) != 1 || result.Sources[0].Name != "Join" {
		t.Fatalf("string result = %#v", result)
	}
	if !strings.Contains(result.CanonicalG1, "00000000000000000000000000009040") || !strings.Contains(result.CanonicalG1, "000000000000000000000000000090c3") {
		t.Fatal("canonical string type or concatenation node missing")
	}
}

func TestIncrementalSessionLiftsOrderedImmutableLocals(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v15/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/locals", Files: map[string]string{
		"locals.go": "package locals\nfunc Scale(left, right int64) int64 {\ntotal := left + right\nscaled := total * 2\nreturn scaled - left\n}\n",
	}})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("local result = %#v", result)
	}
	for _, schema := range []string{"000000000000000000000000000090d0", "000000000000000000000000000090d1", "000000000000000000000000000090d2"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("canonical local schema %s missing", schema)
		}
	}
}

func TestIncrementalSessionLiftsClosedFunctionCalls(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v16/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/calls", Entry: "Render", Files: map[string]string{
		"calls.go": "package calls\nfunc decorate(value string) string { return \"[\" + value + \"]\" }\nfunc combine(left, right string) string { return decorate(left) + decorate(right) }\nfunc Render(left, right string) string { joined := combine(left, right); return decorate(joined) }\n",
	}})
	if !result.Valid || len(result.Diagnostics) != 0 || len(result.Sources) != 3 {
		t.Fatalf("call result = %#v", result)
	}
	if strings.Count(result.CanonicalG1, "00000000000000000000000000009060") != 6 {
		t.Fatal("canonical function calls missing")
	}
}

func TestIncrementalSessionLiftsCompositionalRecords(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v16/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/records", Entry: "Label", Files: map[string]string{
		"records.go": "package records\ntype Item struct { Name string; Enabled bool }\nfunc Label(name string) string { item := Item{Name: name, Enabled: true}; return item.Name }\n",
	}})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("record result = %#v", result)
	}
	for _, schema := range []string{"00000000000000000000000000009030", "00000000000000000000000000009031", "00000000000000000000000000009032", "00000000000000000000000000009033"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("record schema %s missing", schema)
		}
	}
}

func TestIncrementalSessionLiftsMutablePlacesAndWhile(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v18/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/mutation", Entry: "AppendOnce", Files: map[string]string{
		"mutation.go": "package mutation\nfunc AppendOnce(value, suffix string, enabled bool) string { result := value; remaining := enabled; for remaining { result = result + suffix; remaining = false }; return result }\n",
	}})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("mutation result = %#v", result)
	}
	for _, schema := range []string{"000000000000000000000000000090e0", "000000000000000000000000000090e1", "000000000000000000000000000090e2", "000000000000000000000000000090e3", "000000000000000000000000000090e4"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("mutation schema %s missing", schema)
		}
	}
}

func TestIncrementalSessionLiftsIntegerMutationAndWhen(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v19/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/choice", Entry: "Choose", Files: map[string]string{
		"choice.go": "package choice\nfunc Choose(original, replacement int64, enabled bool) int64 { result := original; if enabled { result = replacement }; return result }\n",
	}})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("update rejected: %#v", result.Diagnostics)
	}
	for _, schema := range []string{"000000000000000000000000000090e0", "000000000000000000000000000090e3", "000000000000000000000000000090f0"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("canonical program lacks %s", schema)
		}
	}
}

func TestIncrementalSessionLiftsFoundationEffectInvocation(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v20/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/effect", Entry: "Observe", Files: map[string]string{
		"effect.go": "package effect\nimport \"log\"\nfunc Observe(first, second bool) bool { log.Print(first); log.Print(second); return second }\n",
	}})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("effect result = %#v", result)
	}
	for _, schema := range []string{"000000000000000000000000000090f1", "00000000000000000000000000000015", "00000000000000000000000000000016"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("effect schema %s missing", schema)
		}
	}
	for _, id := range []string{stableID("capability", "observability.log"), stableID("effect", "observability.log")} {
		if strings.Count(result.CanonicalG1, "en "+id+" ") != 1 {
			t.Fatalf("shared declaration %s was not interned exactly once", id)
		}
	}
}

func TestIncrementalSessionLiftsFixedArrayIndexRead(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v21/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/fixed-array", Entry: "Pick", Files: map[string]string{
		"array.go": "package array\nfunc Pick(first, second, third, index int64) int64 { return [3]int64{first, second, third}[index] }\n",
	}})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("array result = %#v", result)
	}
	for _, schema := range []string{"000000000000000000000000000090f2", "000000000000000000000000000090f3", "000000000000000000000000000090f4"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("array schema %s missing", schema)
		}
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
