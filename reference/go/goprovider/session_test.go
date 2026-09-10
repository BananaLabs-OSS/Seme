package goprovider

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestIncrementalSessionPublishesStablePackageMetadata(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v35/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	makeSnapshot := func(revision uint64, reverse bool) DocumentSnapshot {
		files := map[string]string{}
		add := func(name, source string) { files[name] = source }
		entries := [][2]string{
			{"app/main.go", `package app
import "example.test/project/mid"
func Apply(v int64) int64 { return mid.Apply(v) }
func hidden(v int64) int64 { return v }`},
			{"mid/mid.go", `package mid
import "example.test/project/leaf"
func Apply(v int64) int64 { return leaf.Apply(v) }`},
			{"leaf/leaf.go", `package leaf
func Apply(v int64) int64 { return v + 1 }
func private(v int64) int64 { return v }`},
		}
		if reverse {
			for i := len(entries) - 1; i >= 0; i-- {
				add(entries[i][0], entries[i][1])
			}
		} else {
			for _, x := range entries {
				add(x[0], x[1])
			}
		}
		return DocumentSnapshot{Revision: revision, ModulePath: "example.test/project", PackagePath: "example.test/project/app", Entry: "Apply", Files: files}
	}
	a, _ := NewIncrementalSession(module)
	first := a.Apply(makeSnapshot(1, false))
	if !first.Valid {
		t.Fatalf("first=%#v", first)
	}
	b, _ := NewIncrementalSession(module)
	reordered := b.Apply(makeSnapshot(99, true))
	if !reordered.Valid {
		t.Fatalf("reordered=%#v", reordered)
	}
	if !reflect.DeepEqual(first.Packages, reordered.Packages) {
		t.Fatalf("metadata depends on file/client revision ordering\n%#v\n%#v", first.Packages, reordered.Packages)
	}
	if len(first.Packages) != 3 {
		t.Fatalf("packages=%#v", first.Packages)
	}
	byName := map[string]PackageMetadata{}
	for _, p := range first.Packages {
		byName[p.Name] = p
	}
	if !byName["example.test/project/app"].Root || byName["example.test/project/mid"].Root {
		t.Fatal("root marker incorrect")
	}
	if !reflect.DeepEqual(byName["example.test/project/app"].Dependencies, []string{"example.test/project/mid"}) || !reflect.DeepEqual(byName["example.test/project/mid"].Dependencies, []string{"example.test/project/leaf"}) {
		t.Fatalf("dependency closure=%#v", first.Packages)
	}
	ids := map[string]bool{}
	for _, p := range first.Packages {
		if len(p.Functions) != 1 || p.Functions[0].Name != "Apply" {
			t.Fatalf("unexported function leaked: %#v", p.Functions)
		}
		if ids[p.Functions[0].ID] {
			t.Fatal("same-name functions collided")
		}
		ids[p.Functions[0].ID] = true
		if len(p.Functions[0].Parameters) != 1 || p.Functions[0].Result == "" {
			t.Fatal("signature metadata missing")
		}
	}
	first.Packages[0].Dependencies = []string{"mutated"}
	first.Packages[0].Functions[0].Parameters[0] = "mutated"
	bad := makeSnapshot(2, false)
	bad.Files["app/main.go"] = "package app\nfunc Apply("
	retained := a.Apply(bad)
	if retained.Valid || len(retained.Packages) != 3 || reflect.DeepEqual(retained.Packages, first.Packages) {
		t.Fatalf("invalid snapshot did not retain copy-safe metadata: %#v", retained)
	}
}

func TestIncrementalSessionRejectsCrossPackagePrivateMemberWithLocation(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v35/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/private-access", PackagePath: "example.test/private-access/app", Entry: "Apply", Files: map[string]string{
		"model/model.go": "package model\nfunc hidden(v int64) int64 { return v }\nfunc Public(v int64) int64 { return hidden(v) }\n",
		"app/app.go":     "package app\nimport \"example.test/private-access/model\"\nfunc Apply(v int64) int64 { return model.hidden(v) }\n",
	}})
	if !result.Accepted || result.Valid || result.CanonicalG1 != "" || result.LastValidRevision != 0 {
		t.Fatalf("private access result=%#v", result)
	}
	var located bool
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "go.type" && diagnostic.File == "app/app.go" && diagnostic.Line == 3 && diagnostic.Column > 0 && strings.Contains(diagnostic.Message, "hidden") {
			located = true
		}
	}
	if !located {
		t.Fatalf("private access diagnostic=%#v", result.Diagnostics)
	}
}

func TestIncrementalSessionComposesCumulativeTextCollectionFlow(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v30/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	source := `package text
func Sum(values []int64) int64 {
 total := int64(0)
 for _, value := range values { total += value }
 return total
}

func Describe(prefix string, values []int64) string {
 total := Sum(values)
 if total <= 0 { return prefix + ":non-positive" }
 return prefix + ":positive"
}`
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/cumulative-text-collection", Entry: "Describe", Files: map[string]string{"program.go": source}})
	if !result.Valid {
		t.Fatalf("result = %#v", result)
	}
	for _, schema := range []string{"00000000000000000000000000009040", "000000000000000000000000000090c3", "000000000000000000000000000090f7", "00000000000000000000000000009060", "000000000000000000000000000090d0", "000000000000000000000000000090c0"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("canonical graph lacks composed schema %s", schema)
		}
	}
	for schema, want := range map[string]int{
		"000000000000000000000000000090c3": 2,
		"000000000000000000000000000090f7": 1,
		"00000000000000000000000000009060": 1,
		"000000000000000000000000000090d0": 1,
		"000000000000000000000000000090c0": 1,
	} {
		if got := strings.Count(result.CanonicalG1, " "+schema+" 1 "); got != want {
			t.Fatalf("schema %s instance count = %d, want %d", schema, got, want)
		}
	}
}

func TestIncrementalSessionComposesCumulativeStatefulCollectionFlow(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v30/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	source := `package cumulative
type Transition[S, R any] struct { State S; Result R }
type Accumulator struct { Value int64 }
func (state Accumulator) Add(delta int64) Transition[Accumulator, int64] {
 next := Accumulator{Value: state.Value + delta}
 return Transition[Accumulator, int64]{State: next, Result: next.Value}
}

func Sum(values []int64) int64 {
 total := int64(0)
 for _, value := range values { total += value }
 return total
}
func Run(state Accumulator, values []int64, enabled bool) Transition[Accumulator, int64] {
 delta := Sum(values)
 if enabled { return state.Add(delta) }
 return state.Add(0)
}`
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/cumulative-state-flow", Entry: "Run", Files: map[string]string{"program.go": source}})
	if !result.Valid {
		t.Fatalf("result = %#v", result)
	}
	for _, schema := range []string{"000000000000000000000000000090d0", "000000000000000000000000000090c0", "000000000000000000000000000090f7", "00000000000000000000000000009060", "0000000000000000000000000000a003", "0000000000000000000000000000a005"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("canonical graph lacks composed schema %s", schema)
		}
	}
	wantCounts := map[string]int{
		"000000000000000000000000000090d0": 2,
		"000000000000000000000000000090c0": 1,
		"000000000000000000000000000090f7": 1,
		"00000000000000000000000000009060": 1,
		"0000000000000000000000000000a003": 2,
		"0000000000000000000000000000a005": 1,
	}
	for schema, want := range wantCounts {
		if got := strings.Count(result.CanonicalG1, " "+schema+" 1 "); got != want {
			t.Fatalf("schema %s instance count = %d, want %d", schema, got, want)
		}
	}
	invalidSource := strings.Replace(source, "total += value", "total = -value", 1)
	invalid := session.Apply(DocumentSnapshot{Revision: 2, PackagePath: "example.test/cumulative-state-flow", Entry: "Run", Files: map[string]string{"program.go": invalidSource}})
	if !invalid.Accepted || invalid.Valid || invalid.LastValidRevision != 1 || invalid.CanonicalG1 != result.CanonicalG1 {
		t.Fatalf("invalid dependent revision did not retain baseline: %#v", invalid)
	}
	foundIntegrity := false
	for _, diagnostic := range invalid.Diagnostics {
		if diagnostic.Code == "session.call_target_unsupported" {
			foundIntegrity = true
		}
	}
	if !foundIntegrity {
		t.Fatalf("invalid revision diagnostics = %#v", invalid.Diagnostics)
	}
}

func TestIncrementalSessionRejectsCallToOmittedDeclaration(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v30/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/call-integrity", Entry: "Run", Files: map[string]string{"program.go": `package integrity
func Helper(value int64) int64 { return -value }
func Run(value int64) int64 { return Helper(value) }
`}})
	if result.Valid {
		t.Fatal("graph retained a call to an omitted declaration")
	}
	found := false
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "session.call_target_unsupported" {
			found = true
		}
	}
	if !found {
		t.Fatalf("diagnostics = %#v", result.Diagnostics)
	}
}

func TestIncrementalSessionRejectsAmbiguousOptionTupleShape(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v30/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, entry, source, code string
	}{
		{"option tuple", "Find", "package boundary\nfunc Find(value int64) (int64, bool) { return value, true }\n", "session.unsupported_function_shape"},
	} {
		t.Run(test.name, func(t *testing.T) {
			session, err := NewIncrementalSession(module)
			if err != nil {
				t.Fatal(err)
			}
			result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/uab02/" + test.name, Entry: test.entry, Files: map[string]string{"boundary.go": test.source}})
			if result.Valid || len(result.Diagnostics) != 1 {
				t.Fatalf("unsupported boundary accepted: %#v", result)
			}
			diagnostic := result.Diagnostics[0]
			if diagnostic.Code != test.code || diagnostic.File != "boundary.go" || diagnostic.Line != 2 || diagnostic.Column != 1 {
				t.Fatalf("diagnostic = %#v", diagnostic)
			}
		})
	}
}

func TestIncrementalSessionLiftsRuntimeKeyedMapFold(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v30/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	source := `package tally
func Tally(values []int64, key int64) int64 {
 counts := map[int64]int64{}
 for _, value := range values { counts[value] = counts[value] + 1 }
 return counts[key]
}`
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/runtime-map", Entry: "Tally", Files: map[string]string{"tally.go": source}})
	if !result.Valid {
		t.Fatalf("result = %#v", result)
	}
	for _, schema := range []string{"0000000000000000000000000000a040", "0000000000000000000000000000a041", "0000000000000000000000000000a042", "0000000000000000000000000000a043", "000000000000000000000000000090f7"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("canonical graph lacks schema %s", schema)
		}
	}
}

func TestIncrementalSessionLiftsMutableClosureWithExplicitStateThreading(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v29/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	source := `package closure
type Counter func(int64) int64
func MakeCounter(start int64) Counter {
 value := start
 return func(delta int64) int64 { value = value + delta; return value }
}
func Run(start, first, second int64) int64 {
 counter := MakeCounter(start)
 counter(first)
 return counter(second)
}
`
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/mutable-closure", Entry: "Run", Files: map[string]string{"counter.go": source}})
	if !result.Valid {
		t.Fatalf("result = %#v", result)
	}
	for _, schema := range []string{"0000000000000000000000000000a030", "0000000000000000000000000000a031", "0000000000000000000000000000a032", "0000000000000000000000000000a033", "0000000000000000000000000000a034", "0000000000000000000000000000a035", "0000000000000000000000000000a006", "0000000000000000000000000000a007"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("canonical graph lacks schema %s", schema)
		}
	}
	if got := strings.Count(result.CanonicalG1, " 0000000000000000000000000000a035 1 2"); got != 2 {
		t.Fatalf("stateful call count = %d, want 2", got)
	}
}

func TestIncrementalSessionLiftsReturnedImmutableClosure(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v28/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	source := `package closure
type Unary func(int64) int64
func MakeAdder(base int64) Unary { return func(value int64) int64 { return base + value } }
func Apply(fn Unary, value int64) int64 { return fn(value) }
func Run(base, value int64) int64 { return Apply(MakeAdder(base), value) }
`
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/immutable-closure", Entry: "Run", Files: map[string]string{"closure.go": source}})
	if !result.Valid {
		t.Fatalf("result = %#v", result)
	}
	for _, schema := range []string{"0000000000000000000000000000a020", "0000000000000000000000000000a021", "0000000000000000000000000000a022", "0000000000000000000000000000a023", "0000000000000000000000000000a024"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("canonical graph lacks schema %s", schema)
		}
	}
}

func TestIncrementalSessionLiftsInterfaceWitnessesAndDynamicCall(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v27/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	source := `package dispatch
type Adjuster interface { Adjust(value int64) int64 }
type OffsetAdjuster struct { Offset int64 }
func (a OffsetAdjuster) Adjust(value int64) int64 { return value + a.Offset }
type ScaleAdjuster struct { Factor int64 }
func (a ScaleAdjuster) Adjust(value int64) int64 { return value * a.Factor }
func Apply(adjuster Adjuster, value int64) int64 { return adjuster.Adjust(value) }
func Dispatch(useScale bool, amount, value int64) int64 {
 if useScale { return Apply(ScaleAdjuster{Factor: amount}, value) }
 return Apply(OffsetAdjuster{Offset: amount}, value)
}
`
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/interface-dispatch", Entry: "Dispatch", Files: map[string]string{"dispatch.go": source}})
	if !result.Valid {
		t.Fatalf("result = %#v", result)
	}
	for _, schema := range []string{"0000000000000000000000000000a010", "0000000000000000000000000000a011", "0000000000000000000000000000a012", "0000000000000000000000000000a014"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("canonical graph lacks schema %s", schema)
		}
	}
	if got := strings.Count(result.CanonicalG1, " 0000000000000000000000000000a012 1 3"); got != 2 {
		t.Fatalf("satisfaction witness count = %d, want 2", got)
	}
	if got := strings.Count(result.CanonicalG1, " 0000000000000000000000000000a013 1 3"); got != 2 {
		t.Fatalf("interface value count = %d, want 2", got)
	}
	witnessIDs := func(graph string) []string {
		var ids []string
		for _, line := range strings.Split(graph, "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[0] == "en" && fields[2] == "0000000000000000000000000000a012" {
				ids = append(ids, fields[1])
			}
		}
		return ids
	}
	baselineWitnesses := strings.Join(witnessIDs(result.CanonicalG1), ",")
	edited := session.Apply(DocumentSnapshot{Revision: 2, PackagePath: "example.test/interface-dispatch", Entry: "Dispatch", Files: map[string]string{
		"dispatch.go": strings.Replace(source, "return value + a.Offset", "return value - a.Offset", 1),
	}})
	if !edited.Valid || edited.LastValidRevision != 2 || edited.CanonicalG1 == result.CanonicalG1 {
		t.Fatalf("edited interface result = %#v", edited)
	}
	if got := strings.Join(witnessIDs(edited.CanonicalG1), ","); got != baselineWitnesses {
		t.Fatalf("witness identities changed across method-body edit: %s != %s", got, baselineWitnesses)
	}
}

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
	files := map[string]string{
		"decorate.go": "package calls\nfunc decorate(value string) string { return \"[\" + value + \"]\" }\n",
		"combine.go":  "package calls\nfunc combine(left, right string) string { return decorate(left) + decorate(right) }\n",
		"render.go":   "package calls\nfunc Render(left, right string) string { joined := combine(left, right); return decorate(joined) }\n",
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/calls", Entry: "Render", Files: files})
	if !result.Valid || len(result.Diagnostics) != 0 || len(result.Sources) != 3 {
		t.Fatalf("call result = %#v", result)
	}
	if strings.Count(result.CanonicalG1, "00000000000000000000000000009060") != 6 {
		t.Fatal("canonical function calls missing")
	}

	// File-map insertion order is not semantic. A fresh session presented with
	// the same package in another order must produce byte-identical canonical
	// meaning.
	second, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	reordered := second.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/calls", Entry: "Render", Files: map[string]string{
		"render.go": files["render.go"], "decorate.go": files["decorate.go"], "combine.go": files["combine.go"],
	}})
	if !reordered.Valid || reordered.CanonicalG1 != result.CanonicalG1 {
		t.Fatalf("multi-file canonical result depends on presentation order")
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

func TestIncrementalSessionLiftsValueReceiverStateTransition(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v26/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	source := `package counter
type Transition[S, R any] struct { State S; Result R }
type Counter struct { Value int64 }
func (counter Counter) Add(delta int64) Transition[Counter, int64] {
	next := Counter{Value: counter.Value + delta}
	return Transition[Counter, int64]{State: next, Result: next.Value}
}
func Step(counter Counter, delta int64) Transition[Counter, int64] { return counter.Add(delta) }
func Updated(counter Counter, delta int64) Counter { return Step(counter, delta).State }
func Result(counter Counter, delta int64) int64 { return Step(counter, delta).Result }
`
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/counter", Entry: "Step", Files: map[string]string{"counter.go": source}})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("transition result = %#v", result)
	}
	for _, schema := range []string{
		"0000000000000000000000000000a000", "0000000000000000000000000000a001",
		"0000000000000000000000000000a002", "0000000000000000000000000000a003",
		"0000000000000000000000000000a004", "0000000000000000000000000000a005",
		"0000000000000000000000000000a006", "0000000000000000000000000000a007",
	} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("canonical program lacks %s", schema)
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

func TestIncrementalSessionLiftsResultWithPartiallyKeyedNestedRecord(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v36/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/records", PackagePath: "example.test/records", Entry: "Build", Files: map[string]string{
		"records.go": `package records
type Result[T, E any] struct { Ok bool; Value T; Error E }
type Inner struct { Enabled bool; Values []int64 }
type Decision struct { Code int64; Inner Inner; Note string; Marker []byte; Counts map[int64]int64 }
func Build(code int64) Result[Decision, int64] {
	return Result[Decision, int64]{Ok: true, Value: Decision{Code: code}}
}`,
	}})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("partially keyed result record rejected: %#v", result.Diagnostics)
	}
	for _, schema := range []string{"00000000000000000000000000009043", "00000000000000000000000000009033", "0000000000000000000000000000a064", "0000000000000000000000000000a068", "0000000000000000000000000000a041"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("zero-composed record omitted schema %s", schema)
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

func TestIncrementalSessionLiftsFixedArrayParameter(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v21/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/fixed-array-boundary", Entry: "Pick", Files: map[string]string{
		"array.go": "package array\nfunc Pick(values [3]int64, index int64) int64 { return values[index] }\n",
	}})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("array boundary result = %#v", result)
	}
	if strings.Count(result.CanonicalG1, "000000000000000000000000000090f2") < 2 || strings.Count(result.CanonicalG1, "000000000000000000000000000090f4") < 2 {
		t.Fatal("array boundary schemas missing")
	}
}

func TestIncrementalSessionLiftsRangeAsDeterministicFold(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v22/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/fold", Entry: "Sum", Files: map[string]string{
		"fold.go": "package fold\nfunc Sum(values [3]int64) int64 { total := int64(0); for _, value := range values { total += value }; return total }\n",
	}})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("fold result = %#v", result)
	}
	for _, schema := range []string{"000000000000000000000000000090f5", "000000000000000000000000000090f6", "000000000000000000000000000090f7"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("fold schema %s missing", schema)
		}
	}
}

func TestIncrementalSessionLiftsRuntimeSizedSliceFold(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v23/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/slice", Entry: "Sum", Files: map[string]string{
		"slice.go": "package slice\nfunc Sum(values []int64) int64 { total := int64(0); for _, value := range values { total += value }; return total }\n",
	}})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("slice result = %#v", result)
	}
	if !strings.Contains(result.CanonicalG1, "000000000000000000000000000090f8") || !strings.Contains(result.CanonicalG1, "000000000000000000000000000090f7") {
		t.Fatal("slice or fold schema missing")
	}
}

func TestIncrementalSessionLiftsCollectionQueries(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v24/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/query", Entry: "LastOr", Files: map[string]string{
		"query.go": "package query\nfunc LastOr(values []int64, fallback int64) int64 { if len(values) <= 0 { return fallback }; return values[len(values)-1] }\n",
	}})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("query result = %#v", result)
	}
	if !strings.Contains(result.CanonicalG1, "000000000000000000000000000090f9") || !strings.Contains(result.CanonicalG1, "000000000000000000000000000090fa") {
		t.Fatal("collection query schemas missing")
	}
}

func TestIncrementalSessionLiftsImmutableCollectionResults(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v25/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/update", Entry: "UpdateAndAppend", Files: map[string]string{
		"update.go": "package update\nimport \"slices\"\nfunc UpdateAndAppend(values []int64, index int64, replacement int64, appended int64) []int64 { return append(slices.Replace(slices.Clone(values), int(index), int(index)+1, replacement), appended) }\n",
	}})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("update result = %#v", result)
	}
	if !strings.Contains(result.CanonicalG1, "000000000000000000000000000090fb") || !strings.Contains(result.CanonicalG1, "000000000000000000000000000090fc") {
		t.Fatal("collection result schemas missing")
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

func TestIncrementalSessionDeclaresAggregateRecordFields(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v35/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	source := `package aggregate
type State struct { Name string; Values []int64; Counters map[int64]int64 }
func Name(state State) string { return state.Name }
`
	result := session.Apply(DocumentSnapshot{Revision: 1, PackagePath: "example.test/aggregate-record", Entry: "Name", Files: map[string]string{"state.go": source}})
	if !result.Valid {
		t.Fatalf("diagnostics=%#v", result.Diagnostics)
	}
	for _, schema := range []string{"00000000000000000000000000009030", "000000000000000000000000000090f8", "0000000000000000000000000000a040"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("missing schema %s", schema)
		}
	}
}

func TestIncrementalSessionLoadsLocalModulePackageClosure(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v35/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/project", PackagePath: "example.test/project/application", Entry: "Apply", Files: map[string]string{
		"model/value.go":     "package model\nfunc Apply(value int64) int64 { return value + 1 }\n",
		"application/app.go": "package application\nimport \"example.test/project/model\"\nfunc Apply(value int64) int64 { return model.Apply(value) }\n",
	}})
	if !result.Valid {
		t.Fatalf("diagnostics=%#v", result.Diagnostics)
	}
	if strings.Count(result.CanonicalG1, " 00000000000000000000000000009011 1 ") != 2 || !strings.Contains(result.CanonicalG1, " 00000000000000000000000000009060 1 ") {
		t.Fatal("unified imported call graph missing")
	}
	wantEntry := stableID("session-declaration", "example.test/project/application", "Apply")
	if !strings.Contains(result.CanonicalG1, "fi 00000000000000000000000000009151 rf "+wantEntry) {
		t.Fatal("same-named dependency function became the program entry")
	}
}

func TestIncrementalSessionPreservesNestedGenericResultFieldTypes(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v36/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/nested", PackagePath: "example.test/nested/application", Entry: "Revision", Files: map[string]string{
		"model/result.go": `package model
type Result[S any, F any] struct { Ok bool; Value S; Error F }
`,
		"application/read.go": `package application
import "example.test/nested/model"
type State struct { Revision int64 }
type Decision struct { Value State }
func Revision(result model.Result[Decision, int64]) int64 { return result.Value.Value.Revision }
`,
	}})
	if !result.Valid {
		t.Fatalf("diagnostics=%#v", result.Diagnostics)
	}
}

func TestIncrementalSessionSinglePackageMayUseSemanticPathDifferentFromModule(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v30/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{
		Revision: 1, ModulePath: "disk.example/module", PackagePath: "semantic.example/application", Entry: "Run",
		Files: map[string]string{
			"sum.go": "package application\nfunc Sum(left, right int64) int64 { return left + right }\n",
			"run.go": "package application\nfunc Run(left, right int64) int64 { return Sum(left, right) }\n",
		},
	})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("result=%#v", result)
	}
}

func TestIncrementalSessionLiftsCumulativeApplicationPackageClosure(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v35/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, name := range []string{"model/model.go", "policy/policy.go", "application/application.go"} {
		data, readErr := os.ReadFile("../../../fixtures/go-uab-11/" + name)
		if readErr != nil {
			t.Fatal(readErr)
		}
		files[name] = string(data)
	}
	session, err := NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/go-uab-11", PackagePath: "example.test/go-uab-11/application", Entry: "Apply", Files: files})
	if !result.Valid || len(result.Diagnostics) != 0 {
		t.Fatalf("result=%#v", result)
	}
	for _, schema := range []string{"0000000000000000000000000000a044", "0000000000000000000000000000a062", "000000000000000000000000000090f1"} {
		if !strings.Contains(result.CanonicalG1, schema) {
			t.Fatalf("missing cumulative schema %s", schema)
		}
	}
	if len(result.Sources) < 8 {
		t.Fatalf("package closure sources=%d", len(result.Sources))
	}
}
