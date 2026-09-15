package goprojector_test

import (
	"bytes"
	"go/importer"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"seme.local/reference/goprojector"
	"seme.local/reference/goprovider"
)

func TestProjectionEnvelopeRequiresExactIndependentReprojection(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v13/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/envelope", Entry: "Add", Files: map[string]string{"program.go": "package envelope\nfunc Add(value int64) int64 { return value + 1 }\n"}})
	if !result.Valid {
		t.Fatalf("lift=%#v", result.Diagnostics)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "envelope")
	if err != nil {
		t.Fatal(err)
	}
	verified, present, err := goprojector.VerifyProjectionEnvelope(projected)
	if err != nil || !present || !bytes.Equal(verified, []byte(result.CanonicalG1)) {
		t.Fatalf("verify present=%v err=%v", present, err)
	}
	changedSource := bytes.Replace(projected, []byte("value + 1"), []byte("value + 2"), 1)
	if _, present, err := goprojector.VerifyProjectionEnvelope(changedSource); !present || err == nil {
		t.Fatal("semantic source edit retained projection identity")
	}
	changedDigest := append([]byte(nil), projected...)
	digestMarker := bytes.Index(changedDigest, []byte("//seme:projection-v1 ")) + len("//seme:projection-v1 ")
	if digestMarker < len("//seme:projection-v1 ") {
		t.Fatal("digest envelope missing")
	}
	if changedDigest[digestMarker] == 'a' {
		changedDigest[digestMarker] = 'b'
	} else {
		changedDigest[digestMarker] = 'a'
	}
	if _, present, err := goprojector.VerifyProjectionEnvelope(changedDigest); !present || err == nil {
		t.Fatal("forged digest accepted")
	}
	changedEnvelope := append([]byte(nil), projected...)
	marker := bytes.Index(changedEnvelope, []byte("//seme:graph "))
	if marker < 0 {
		t.Fatal("graph envelope missing")
	}
	position := marker + len("//seme:graph ")
	if changedEnvelope[position] == 'A' {
		changedEnvelope[position] = 'B'
	} else {
		changedEnvelope[position] = 'A'
	}
	if _, present, err := goprojector.VerifyProjectionEnvelope(changedEnvelope); !present || err == nil {
		t.Fatal("forged graph payload accepted")
	}

	different := session.Apply(goprovider.DocumentSnapshot{Revision: 2, PackagePath: "example.test/envelope", Entry: "Add", Files: map[string]string{"program.go": "package envelope\nfunc Add(value int64) int64 { return value + 2 }\n"}})
	if !different.Valid {
		t.Fatalf("different lift=%#v", different.Diagnostics)
	}
	differentProjection, err := goprojector.Project([]byte(different.CanonicalG1), "envelope")
	if err != nil {
		t.Fatal(err)
	}
	envelopeBounds := func(source []byte) (int, int) {
		start := bytes.Index(source, []byte("//seme:projection-v1 "))
		if start < 0 {
			return -1, -1
		}
		endRelative := bytes.Index(source[start:], []byte("\n\n"))
		if endRelative < 0 {
			return -1, -1
		}
		return start, start + endRelative
	}
	leftStart, leftEnd := envelopeBounds(projected)
	rightStart, rightEnd := envelopeBounds(differentProjection)
	if leftStart < 0 || rightStart < 0 {
		t.Fatal("projection bounds")
	}
	validDifferentGraphClaimingOldSource := append([]byte(nil), projected[:leftStart]...)
	validDifferentGraphClaimingOldSource = append(validDifferentGraphClaimingOldSource, differentProjection[rightStart:rightEnd]...)
	validDifferentGraphClaimingOldSource = append(validDifferentGraphClaimingOldSource, projected[leftEnd:]...)
	if _, present, err := goprojector.VerifyProjectionEnvelope(validDifferentGraphClaimingOldSource); !present || err == nil {
		t.Fatal("valid but different graph retained old source identity")
	}
}

func TestProjectsGoInterfaceDispatchAndReliftsByteIdentically(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v27/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("../../../fixtures/go-execution-v27/adjuster.go")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/go-uab05-project", Entry: "Dispatch", Files: map[string]string{"adjuster.go": string(source)}})
	if !first.Valid {
		t.Fatalf("lift: %#v", first.Diagnostics)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "interfacedispatch")
	if err != nil {
		t.Fatal(err)
	}
	secondSession, _ := goprovider.NewIncrementalSession(module)
	second := secondSession.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/go-uab05-project", Entry: "Dispatch", Files: map[string]string{"adjuster.go": string(projected)}})
	if !second.Valid {
		t.Fatalf("relift: %#v\n%s", second.Diagnostics, projected)
	}
	if !bytes.Equal([]byte(first.CanonicalG1), []byte(second.CanonicalG1)) {
		t.Fatal("projection did not relift byte-identically")
	}
}

func TestProjectsTypedMultiFileCallGraphAndRelifts(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v16/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"decorate.go": "package nativeproof\nfunc decorate(value string) string { return \"[\" + value + \"]\" }\n",
		"combine.go":  "package nativeproof\nfunc combine(left, right string) string { return decorate(left) + decorate(right) }\n",
		"render.go":   "package nativeproof\nfunc Render(left, right string) string { return decorate(combine(left, right)) }\n",
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/go-uab-call", Entry: "Render", Files: files})
	if !first.Valid {
		t.Fatalf("initial lift: %#v", first.Diagnostics)
	}
	runNative(t, files)
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"func decorate(value string) string", "func combine(left string, right string) string", "func Render(left string, right string) string"} {
		if !strings.Contains(string(projected), fragment) {
			t.Fatalf("projection lacks %q:\n%s", fragment, projected)
		}
	}

	secondSession, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	second := secondSession.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/go-uab-call", Entry: "Render", Files: map[string]string{"projected.go": string(projected)}})
	if !second.Valid {
		t.Fatalf("projected re-lift: %#v\n%s", second.Diagnostics, projected)
	}
	if second.CanonicalG1 != first.CanonicalG1 {
		t.Fatal("projected Go did not re-lift to byte-identical canonical meaning")
	}

	runNative(t, map[string]string{"projected.go": string(projected)})
}

func TestProjectsNativeDeferBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v61/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-defer", Entry: "Run", Files: map[string]string{
		"defer.go": "package sample\nfunc Finish(value int64) int64 { return value }\nfunc Run(value int64) int64 { defer Finish(value); return value }\n",
	}})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(projected), "defer Finish(value)") {
		t.Fatalf("projection lacks defer:\n%s", projected)
	}
}

func TestProjectsNativeFieldAssignmentBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v63/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-field-assignment", Entry: "Set", Files: map[string]string{
		"field.go": "package sample\ntype State struct { Name string }\nfunc Set(state *State, name string) { state.Name = name }\n",
	}})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "sample")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(projected), "state.Name = name") {
		t.Fatalf("projection lacks field assignment:\n%s", projected)
	}
}

func TestProjectsNativeVariadicExpansionBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v89/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-ellipsis", Entry: "Expand", Files: map[string]string{
		"ellipsis.go": "package sample\nimport \"fmt\"\nfunc Expand(values []any) string { return fmt.Sprint(values...) }\n",
	}, ExternalImporter: importer.Default()})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "sample")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(projected), "fmt.Sprint(values...)") {
		t.Fatalf("projection lacks variadic expansion:\n%s", projected)
	}
}

func TestProjectsParallelFieldAssignmentWithGoOrdering(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v90/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/parallel-fields", Entry: "Swap", Files: map[string]string{
		"swap.go": "package sample\ntype Pair struct { Left, Right string }\nfunc Swap(pair *Pair) { pair.Left, pair.Right = pair.Right, pair.Left }\n",
	}})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	runNativeWithTest(t, map[string]string{"projected.go": string(projected)}, `func TestNative(t *testing.T) {
	value := &Pair{Left: "left", Right: "right"}
	Swap(value)
	if value.Left != "right" || value.Right != "left" { t.Fatal(value) }
}`)
}

func TestProjectsImportedPackageBindingBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v91/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/package-binding", Entry: "Output", Files: map[string]string{
		"output.go": "package sample\nimport \"os\"\nfunc Output() *os.File { return os.Stdout }\n",
	}, ExternalImporter: importer.Default()})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(projected), "return os.Stdout") {
		t.Fatalf("projection lacks imported package binding:\n%s", projected)
	}
	runNativeWithTest(t, map[string]string{"projected.go": string(projected)}, `func TestNative(t *testing.T) { if Output() == nil { t.Fatal("stdout") } }`)
}

func TestProjectsExplicitNativeConversionsBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v92/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/conversion", Entry: "Text", Files: map[string]string{
		"conversion.go": "package sample\nfunc Text(value []byte) string { return string(value) }\n",
	}})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(projected), "return string(value)") {
		t.Fatalf("projection lacks native conversion:\n%s", projected)
	}
	runNativeWithTest(t, map[string]string{"projected.go": string(projected)}, `func TestNative(t *testing.T) { if Text([]byte("snow")) != "snow" { t.Fatal("text") } }`)
}

func TestProjectsCharacterLiteralInTypedCallBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v93/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/character", Entry: "Line", Files: map[string]string{
		"character.go": "package sample\nfunc Line(value []byte) []byte { return append(value, '\\n') }\n",
	}})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	runNativeWithTest(t, map[string]string{"projected.go": string(projected)}, `func TestNative(t *testing.T) {
	got := Line([]byte("row"))
	if string(got) != "row\n" { t.Fatalf("%q", got) }
}`)
}

func TestProjectsPackageBindingAssignmentBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v94/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/binding", Entry: "Enable", Files: map[string]string{
		"binding.go": "package sample\nvar Enabled bool\nfunc Enable(value bool) { Enabled = value }\n",
	}})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(projected), "Enabled = value") {
		t.Fatalf("projection lacks package binding assignment:\n%s", projected)
	}
	runNativeWithTest(t, map[string]string{"projected.go": string(projected), "binding.go": "package nativeproof\nvar Enabled bool\n"}, `func TestNative(t *testing.T) { Enable(true); if !Enabled { t.Fatal("binding") } }`)
}

func TestProjectsPointerAssignmentBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v95/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/pointer", Entry: "Set", Files: map[string]string{
		"pointer.go": "package sample\nfunc Set(target *string, value string) { *target = value }\n",
	}})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(projected), "*(target) = value") {
		t.Fatalf("projection lacks pointer assignment:\n%s", projected)
	}
	runNativeWithTest(t, map[string]string{"projected.go": string(projected)}, `func TestNative(t *testing.T) { value := "before"; Set(&value, "after"); if value != "after" { t.Fatal(value) } }`)
}

func TestProjectsMutableParameterThroughCanonicalPlace(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v64/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/mutable-parameter", Entry: "Normalize", Files: map[string]string{
		"parameter.go": "package sample\nimport \"strings\"\nfunc Normalize(value string) string { value = strings.TrimSpace(value); return value }\n",
	}, ExternalImporter: importer.Default()})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "sample")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"seme_mutable_value := value", "seme_mutable_value = strings.TrimSpace(seme_mutable_value)", "return seme_mutable_value"} {
		if !strings.Contains(string(projected), fragment) {
			t.Fatalf("projection lacks %q:\n%s", fragment, projected)
		}
	}
}

func TestProjectsTypedNativeSwitchBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v65/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-switch", Entry: "Select", Files: map[string]string{
		"switch.go": "package sample\nfunc Select(value string) string { switch value { case \"a\", \"b\": return \"match\"; default: return \"other\" } }\n",
	}})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "sample")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"switch value {", `case "a", "b":`, "default:"} {
		if !strings.Contains(string(projected), fragment) {
			t.Fatalf("projection lacks %q:\n%s", fragment, projected)
		}
	}
}

func TestProjectsNearestLoopBranchBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v66/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-branch", Entry: "Run", Files: map[string]string{
		"branch.go": "package sample\nfunc Run(enabled bool) bool { remaining := enabled; for remaining { remaining = false; continue }; return remaining }\n",
	}})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "sample")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(projected), "continue") {
		t.Fatalf("projection lacks continue:\n%s", projected)
	}
}

func TestProjectsNativeAddressBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v67/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-address", Entry: "Address", Files: map[string]string{
		"address.go": "package sample\nfunc Address(value int64) *int64 { return &value }\n",
	}})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "sample")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(projected), "return &(value)") {
		t.Fatalf("projection lacks address expression:\n%s", projected)
	}
}

func TestProjectsNativeLengthBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v68/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-length", Entry: "Size", Files: map[string]string{
		"length.go": "package sample\nfunc Size(values map[string]int) int { return len(values) }\n",
	}})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "sample")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(projected), "return len(") || !strings.Contains(string(projected), "values") {
		t.Fatalf("projection lacks native length:\n%s", projected)
	}
}

func TestProjectsNativeCollectionBuiltinsBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v68/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-builtins", Entry: "Build", Files: map[string]string{
		"builtins.go": "package sample\ntype Item struct { Value int64 }\nfunc Build(values []Item, value Item) []Item { out := make([]Item, 0, 1); out = append(out, values...); out = append(out, value); return out }\n",
	}})
	if !first.Valid || len(first.NativeIslands) != 0 {
		t.Fatalf("initial lift: %#v", first)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	for _, fragment := range []string{"make([]Item", "append(", "values)...", "Item(value)"} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("projection lacks %q:\n%s", fragment, source)
		}
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) {
	got := Build([]Item{{Value: 1}}, Item{Value: 2})
	if len(got) != 2 || got[0].Value != 1 || got[1].Value != 2 { t.Fatal(got) }
}`)
}

func TestProjectsNativeMapRangeBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v70/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-map-range", Entry: "Total", Files: map[string]string{
		"range.go": "package sample\nfunc Total(values map[string]int64) int64 { total := int64(0); for _, value := range values { total = total + value }; return total }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("native map range lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	if !strings.Contains(source, "for _, value := range values") {
		t.Fatalf("projection lacks map range:\n%s", source)
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { if Total(map[string]int64{"a": 2, "b": 3}) != 5 { t.Fatal("range") } }`)
}

func TestProjectsNativeStringRangeBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v96/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-string-range", Entry: "Count", Files: map[string]string{
		"range.go": "package sample\nfunc Count(value string) int64 { total := int64(0); for index, codepoint := range value { if index >= 0 && codepoint > 0 { total = total + 1 } }; return total }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("native string range lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	if !strings.Contains(source, "for index, codepoint := range value") {
		t.Fatalf("projection lacks string range:\n%s", source)
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { if Count("a界") != 2 { t.Fatal("range") } }`)
}

func TestProjectsMapLookupIfInitializerBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v97/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/map-if-initializer", Entry: "Lookup", Files: map[string]string{
		"lookup.go": "package sample\nfunc Lookup(values map[string]string, key string) string { if value, ok := values[key]; ok { return value }; return \"missing\" }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("map lookup if initializer lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	for _, fragment := range []string{"(map[string]string)(values)[string(key)]", "if ok", "return value"} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("projection lacks %q:\n%s", fragment, source)
		}
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { if Lookup(map[string]string{"a":"yes"}, "a") != "yes" || Lookup(nil, "x") != "missing" { t.Fatal("lookup") } }`)
}

func TestProjectsNativeIntegerForBindingBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v98/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-for-binding", Entry: "Sum", Files: map[string]string{
		"loop.go": "package sample\nfunc Sum(limit int) int { total := 0; for index := 0; index < limit; index++ { total += index }; return total }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("native integer for binding lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	for _, fragment := range []string{"index := int(0)", "for !(limit <= index)", "index = (index + 1)"} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("projection lacks %q:\n%s", fragment, source)
		}
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { if Sum(5) != 10 { t.Fatal(Sum(5)) } }`)
}

func TestProjectsNativeConcurrentStartBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v99/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-concurrent-start", Entry: "Start", Files: map[string]string{
		"start.go": "package sample\nfunc Work(value int64) {}\nfunc Start(value int64) { go Work(value) }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("native concurrent start lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	if !strings.Contains(source, "go Work(value)") {
		t.Fatalf("projection lacks concurrent start:\n%s", source)
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { Start(1) }`)
}

func TestProjectsNativeMethodValueBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v105/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "nativeproof", Entry: "Bind", Files: map[string]string{
		"binding.go": "package sample\ntype Router struct{ Base int64 }\nfunc (r *Router) Resolve(value int64) int64 { return r.Base + value }\nfunc Bind(r *Router) func(int64) int64 { return r.Resolve }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("native method value lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	if !strings.Contains(source, "return (r).Resolve") {
		t.Fatalf("projection lacks method value:\n%s", source)
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { if Bind(&Router{Base: 1})(2) != 3 { t.Fatal("method value") } }`)
}

func TestProjectsVariadicNativeMethodCallBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v106/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "nativeproof", Entry: "Run", Files: map[string]string{
		"method.go": "package sample\ntype Runner struct{ Base int64 }\nfunc (r *Runner) Sum(values ...int64) int64 { return r.Base + values[0] }\nfunc Run(r *Runner, values []int64) int64 { return r.Sum(values...) }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("variadic native method lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	if !strings.Contains(source, "r.Sum(values...)") {
		t.Fatalf("projection lacks variadic method call:\n%s", source)
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { if Run(&Runner{Base: 1}, []int64{2}) != 3 { t.Fatal("variadic method") } }`)
}

func TestProjectsNativeIndirectInvocationBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v107/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "nativeproof", Entry: "Run", Files: map[string]string{
		"call.go": "package sample\nfunc Run(cancel func()) { cancel() }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("native indirect invocation lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	if !strings.Contains(source, "cancel()") {
		t.Fatalf("projection lacks native indirect invocation:\n%s", source)
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { called := false; Run(func() { called = true }); if !called { t.Fatal("not called") } }`)
}

func TestProjectsCompositeTypeConversionBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v108/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "nativeproof", Entry: "Run", Files: map[string]string{
		"conversion.go": "package sample\nfunc Run(text string) []byte { return []byte(text) }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("composite type conversion lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	if !strings.Contains(source, "([]byte)(text)") {
		t.Fatalf("projection lacks composite type conversion:\n%s", source)
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { if string(Run("seme")) != "seme" { t.Fatal("conversion") } }`)
}

func TestProjectsGeneralImmutableClosureBlockBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v110/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "nativeproof", Entry: "Run", Files: map[string]string{
		"closure.go": "package sample\nfunc Run(base int64) func(int64) int64 { return func(value int64) int64 { total := base + value; return total } }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("general closure lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	if !strings.Contains(source, "func(value int64) int64") || !strings.Contains(source, "total := (base + value)") {
		t.Fatalf("projection lacks general closure block:\n%s", source)
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { if Run(4)(5) != 9 { t.Fatal("closure") } }`)
}

func TestProjectsNativeChannelSendBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v100/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-channel-send", Entry: "Send", Files: map[string]string{
		"send.go": "package sample\nfunc Send(done chan struct{}) { done <- struct{}{} }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("native channel send lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	if !strings.Contains(source, "done <- struct{}{}") {
		t.Fatalf("projection lacks channel send:\n%s", source)
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { done := make(chan struct{}, 1); Send(done); <-done }`)
}

func TestProjectsNativeSelectBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v101/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-select", Entry: "Wait", Files: map[string]string{
		"select.go": "package sample\nfunc Wait(done, tick chan struct{}) int64 { for { select { case <-done: return 1; case tick <- struct{}{}: return 2; default: return 3 } } }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("native select lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	for _, fragment := range []string{"select {", "case <-done:", "case tick <- struct{}{}:", "default:"} {
		if !strings.Contains(source, fragment) {
			t.Fatalf("projection lacks %q:\n%s", fragment, source)
		}
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { done := make(chan struct{}); var tick chan struct{}; if Wait(done, tick) != 3 { t.Fatal("default") } }`)
}

func TestProjectsParallelClassicForBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v102/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/parallel-for", Entry: "Reverse", Files: map[string]string{
		"reverse.go": "package sample\nfunc Reverse(values []int64) []int64 { for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 { values[i], values[j] = values[j], values[i] }; return values }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("parallel classic for lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	if !strings.Contains(source, "for ") {
		t.Fatalf("projection lacks loop:\n%s", source)
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { got := Reverse([]int64{1,2,3,4}); want := []int64{4,3,2,1}; for i := range want { if got[i] != want[i] { t.Fatalf("%v", got) } } }`)
}

func TestProjectsNativeChannelReceiveBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v103/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-channel-receive", Entry: "Wait", Files: map[string]string{
		"receive.go": "package sample\nfunc Wait(done chan struct{}) { <-done }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("native channel receive lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	if !strings.Contains(source, "<-done") {
		t.Fatalf("projection lacks channel receive:\n%s", source)
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { done := make(chan struct{}, 1); done <- struct{}{}; Wait(done) }`)
}

func TestProjectsEmptyRecordLiteralBackToGo(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v104/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/empty-record", Entry: "Empty", Files: map[string]string{
		"empty.go": "package sample\ntype Entry struct { Count int64; Name string }\nfunc Empty() Entry { return Entry{} }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("empty record lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { got := Empty(); if got.Count != 0 || got.Name != "" { t.Fatalf("%+v", got) } }`)
}

func TestProjectsNativeSliceBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v71/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-slice", Entry: "Window", Files: map[string]string{
		"slice.go": "package sample\nfunc Window(values []int64, low int, high int) []int64 { return values[low:high] }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("native slice lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	if !strings.Contains(source, "values[int(low):int(high)]") {
		t.Fatalf("projection lacks native slice:\n%s", source)
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { got := Window([]int64{1,2,3}, 1, 3); if len(got)!=2 || got[0]!=2 || got[1]!=3 { t.Fatal(got) } }`)
}

func TestProjectsNativeDereferenceBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v72/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-dereference", Entry: "Value", Files: map[string]string{
		"dereference.go": "package sample\nfunc Value(pointer *int64) int64 { return *pointer }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("native dereference lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	if !strings.Contains(source, "return *(pointer)") {
		t.Fatalf("projection lacks native dereference:\n%s", source)
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { value := int64(9); if Value(&value) != 9 { t.Fatal("dereference") } }`)
}

func TestProjectsNativeBinaryBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v73/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-binary", Entry: "Mix", Files: map[string]string{
		"binary.go": "package sample\nfunc Mix(value uint64, shift uint) uint64 { return (value << shift) | (value % 3) }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("native binary lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	for _, operator := range []string{" << ", " | ", " % "} {
		if !strings.Contains(source, operator) {
			t.Fatalf("projection lacks %q:\n%s", operator, source)
		}
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { if Mix(5, 2) != 22 { t.Fatal(Mix(5, 2)) } }`)
}

func TestProjectsNativeIndexAssignmentBackToGoSyntax(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v74/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-index-assignment", Entry: "Set", Files: map[string]string{
		"index.go": "package sample\nfunc Set(values map[string]int64, key string, value int64) int64 { values[key] = value; return values[key] }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("native index assignment lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	source := string(projected)
	if !strings.Contains(source, "values[key] = value") {
		t.Fatalf("projection lacks native index assignment:\n%s", source)
	}
	runNativeWithTest(t, map[string]string{"projected.go": source}, `func TestNative(t *testing.T) { values:=map[string]int64{}; if Set(values,"x",7)!=7 { t.Fatal(values) } }`)
}

func TestProjectsCompoundAssignmentAsCanonicalMutation(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v75/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-compound", Entry: "Add", Files: map[string]string{
		"compound.go": "package sample\nfunc Add(value int64) int64 { total := value; total += 2; return total }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("compound lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	runNativeWithTest(t, map[string]string{"projected.go": string(projected)}, `func TestNative(t *testing.T) { if Add(5)!=7 { t.Fatal(Add(5)) } }`)
}

func TestProjectsNativeMultiResultAndIfInitializerBindings(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v76/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	source := "package sample\nfunc Pair(value int64) (int64, *int64) { return value, &value }\nfunc Read(value int64) int64 { first, pointer := Pair(value); if current := pointer; current == pointer { return first }; return 0 }\n"
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/native-bindings", Entry: "Read", Files: map[string]string{"bindings.go": source}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("native bindings lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	runNativeWithTest(t, map[string]string{"projected.go": string(projected)}, `func TestNative(t *testing.T) { if Read(11)!=11 { t.Fatal(Read(11)) } }`)
}

func TestProjectsPredeclaredErrorMethod(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v77/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/predeclared-method", Entry: "Text", Files: map[string]string{
		"method.go": "package sample\nfunc Text(failure error) string { return failure.Error() }\n",
	}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("predeclared method lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(projected), "return failure.Error()") {
		t.Fatalf("projection lacks native method call:\n%s", projected)
	}
	runNativeWithTest(t, map[string]string{"projected.go": string(projected)}, `type failure string
func (f failure) Error() string { return string(f) }
func TestNative(t *testing.T) { if Text(failure("broken")) != "broken" { t.Fatal("method") } }`)
}

func TestProjectsMixedProductShortDeclaration(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v78/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	source := "package sample\nfunc Pair(value int64, failure error) (int64, error) { return value, failure }\nfunc Read(value int64, failure error) int64 { err := failure; first, err := Pair(value, err); if err != nil { return 0 }; return first }\n"
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/mixed-short", Entry: "Read", Files: map[string]string{"mixed.go": source}})
	if !result.Valid || len(result.NativeIslands) != 0 {
		t.Fatalf("mixed short declaration lift = %#v", result)
	}
	projected, err := goprojector.Project([]byte(result.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(projected), "first, err := Pair(value, err)") {
		t.Fatalf("projection lacks mixed short declaration:\n%s", projected)
	}
	runNativeWithTest(t, map[string]string{"projected.go": string(projected)}, `func TestNative(t *testing.T) { if Read(12, nil) != 12 { t.Fatal("mixed") } }`)
}

func runNative(t *testing.T, files map[string]string) {
	runNativeWithTest(t, files, "func TestNative(t *testing.T) { if got := Render(\"a\", \"b\"); got != \"[[a][b]]\" { t.Fatal(got) } }")
}

func runNativeWithTest(t *testing.T, files map[string]string, body string) {
	t.Helper()
	dir := t.TempDir()
	// The generated projection uses no edition-specific syntax. Keeping this
	// disposable native-oracle module at Go 1.25 lets both the repository's
	// declared 1.26 toolchain and the local fallback toolchain execute it.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.test/go-uab-call\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	testSource := "package nativeproof\nimport \"testing\"\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "projected_test.go"), []byte(testSource), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "-buildvcs=false", "./...")
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("native projected Go failed: %v\n%s", err, output)
	}
}

func TestProjectsExistingCompositeCollectionsAndRecords(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v30/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, fixture, entry, nativeTest string }{
		{"record", "../../../fixtures/go-execution-v17/function.go", "Label", `func TestNative(t *testing.T) { if Label("雪🦀") != "雪🦀" { t.Fatal("record") } }`},
		{"array", "../../../fixtures/go-execution-v21/function.go", "Pick", `func TestNative(t *testing.T) {
 if Pick(-7, 0, 42, 0) != -7 || Pick(-7, 0, 42, 1) != 0 || Pick(-7, 0, 42, 2) != 42 { t.Fatal("array") }
 for _, index := range []int64{-1, 3} { func() { defer func() { if recover() == nil { t.Fatalf("index %d did not panic", index) } }(); Pick(1, 2, 3, index) }() }
}`},
		{"slice", "../../../fixtures/go-execution-v23/function.go", "Sum", `func TestNative(t *testing.T) {
 if Sum(nil) != 0 || Sum([]int64{-7, 0, 42}) != 35 || Sum([]int64{1, 2, 3, 4, 5}) != 15 || Sum([]int64{9223372036854775807, 1, -1}) != 9223372036854775807 { t.Fatal("slice") }
}`},
		{"map", "../../../fixtures/go-execution-v30/tally.go", "Tally", `func TestNative(t *testing.T) {
 values := []int64{3, -2, 3, 7, -2, 3}
 if Tally(values, 3) != 3 || Tally(values, -2) != 2 || Tally([]int64{3, -2, 3}, 99) != 0 || Tally([]int64{-2, 3, 3, -2, 7, 3}, 3) != 3 || Tally(nil, 1) != 0 { t.Fatal("map") }
 maximum := make([]int64, 512); for i := range maximum { maximum[i] = -9 }; if Tally(maximum, -9) != 512 { t.Fatal("map maximum") }
}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, err := os.ReadFile(test.fixture)
			if err != nil {
				t.Fatal(err)
			}
			session, err := goprovider.NewIncrementalSession(module)
			if err != nil {
				t.Fatal(err)
			}
			first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/go-uab02/" + test.name, Entry: test.entry, Files: map[string]string{"original.go": string(source)}})
			if !first.Valid {
				t.Fatalf("initial lift: %#v", first.Diagnostics)
			}
			projected, err := goprojector.Project([]byte(first.CanonicalG1), "nativeproof")
			if err != nil {
				t.Fatal(err)
			}
			secondSession, err := goprovider.NewIncrementalSession(module)
			if err != nil {
				t.Fatal(err)
			}
			second := secondSession.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/go-uab02/" + test.name, Entry: test.entry, Files: map[string]string{"projected.go": string(projected)}})
			if !second.Valid {
				t.Fatalf("projected re-lift: %#v\n%s", second.Diagnostics, projected)
			}
			if second.CanonicalG1 != first.CanonicalG1 {
				t.Fatalf("%s projection changed canonical meaning", test.name)
			}
			runNativeWithTest(t, map[string]string{"projected.go": string(projected)}, test.nativeTest)
		})
	}
}

func TestRejectsMalformedCompositeSemantics(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v30/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, fixture, entry, old, replacement, diagnostic string }{
		{"record field order", "../../../fixtures/go-execution-v17/function.go", "Label", "fi 00000000000000000000000000009312 uu 0", "fi 00000000000000000000000000009312 uu 1", "go_projection.record_field_order"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, err := os.ReadFile(test.fixture)
			if err != nil {
				t.Fatal(err)
			}
			session, err := goprovider.NewIncrementalSession(module)
			if err != nil {
				t.Fatal(err)
			}
			result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/go-uab02/reject/" + test.name, Entry: test.entry, Files: map[string]string{"original.go": string(source)}})
			if !result.Valid {
				t.Fatal(result.Diagnostics)
			}
			mutated := strings.Replace(result.CanonicalG1, test.old, test.replacement, 1)
			if mutated == result.CanonicalG1 {
				t.Fatal("mutation did not apply")
			}
			if _, err := goprojector.Project([]byte(mutated), "nativeproof"); err == nil || !strings.Contains(err.Error(), test.diagnostic) {
				t.Fatalf("malformed composite accepted: %v", err)
			}
		})
	}
}

func TestProjectsBytesOptionAndResultConstructors(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v32/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	source := `package tagged
type Option[T any] struct { Some bool; Value T }
type Result[T, E any] struct { Ok bool; Value T; Error E }
func Some(value int64) Option[int64] { return Option[int64]{Some: true, Value: value} }
func None() Option[int64] { return Option[int64]{} }
func Accepted(value int64) Result[int64, string] { return Result[int64, string]{Ok: true, Value: value} }
func Rejected(message string) Result[int64, string] { return Result[int64, string]{Ok: false, Error: message} }
func Evidence() []byte { return []byte{0, 127, 255} }
func OptionValue(option Option[int64], fallback int64) int64 { if option.Some { return option.Value }; return fallback }
func ResultMessage(result Result[int64, string]) string { if result.Ok { return "ok" }; return result.Error }
`
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/go-uab02/tagged", Entry: "Evidence", Files: map[string]string{"tagged.go": source}})
	if !first.Valid {
		t.Fatalf("initial lift: %#v", first.Diagnostics)
	}
	for _, schema := range []string{sBytesLiteralForTest, sOptionNoneForTest, sOptionSomeForTest, sResultOkForTest, sResultErrorForTest, sOptionMatchForTest, sResultMatchForTest} {
		if !strings.Contains(first.CanonicalG1, schema) {
			t.Fatalf("canonical graph lacks %s", schema)
		}
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	secondSession, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	second := secondSession.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/go-uab02/tagged", Entry: "Evidence", Files: map[string]string{"projected.go": string(projected)}})
	if !second.Valid {
		t.Fatalf("projected re-lift: %#v\n%s", second.Diagnostics, projected)
	}
	if second.CanonicalG1 != first.CanonicalG1 {
		t.Fatal("tagged projection changed canonical meaning")
	}
	runNativeWithTest(t, map[string]string{"projected.go": string(projected)}, `func TestNative(t *testing.T) {
 if got := Some(7); !got.Some || got.Value != 7 { t.Fatal(got) }
 if None().Some { t.Fatal("none") }
 if got := Accepted(9); !got.Ok || got.Value != 9 { t.Fatal(got) }
 if got := Rejected("no"); got.Ok || got.Error != "no" { t.Fatal(got) }
 if got := Evidence(); len(got) != 3 || got[0] != 0 || got[2] != 255 { t.Fatal(got) }
 if OptionValue(Some(7), 3) != 7 || OptionValue(None(), 3) != 3 { t.Fatal("option match") }
 if ResultMessage(Accepted(9)) != "ok" || ResultMessage(Rejected("no")) != "no" { t.Fatal("result match") }
}`)
}

const (
	sBytesLiteralForTest = "0000000000000000000000000000a064"
	sOptionNoneForTest   = "0000000000000000000000000000a051"
	sOptionSomeForTest   = "0000000000000000000000000000a052"
	sResultOkForTest     = "00000000000000000000000000009043"
	sResultErrorForTest  = "00000000000000000000000000009044"
	sOptionMatchForTest  = "0000000000000000000000000000a063"
	sResultMatchForTest  = "0000000000000000000000000000a062"
)

func TestRejectsUnsupportedCanonicalExpression(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v16/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/go-uab-reject", Entry: "Add", Files: map[string]string{
		"add.go": "package reject\nfunc Add(left, right int64) int64 { return left + right }\n",
	}})
	if !result.Valid {
		t.Fatal(result.Diagnostics)
	}
	unsupported := strings.Replace(result.CanonicalG1, "00000000000000000000000000009014 1 3", "0000000000000000000000000000ffff 1 3", 1)
	if _, err := goprojector.Project([]byte(unsupported), "reject"); err == nil || !strings.Contains(err.Error(), "go_projection.unsupported_expression") {
		t.Fatalf("unsupported semantic expression accepted: %v", err)
	}
}

func TestProjectsScopedRecordTypeInsideFunction(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v81/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	source := `package scoped
func Read(value int64) int64 {
	type local struct { Value int64 }
	item := local{Value: value}
	return item.Value
}
`
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(goprovider.DocumentSnapshot{Revision: 1, PackagePath: "example.test/scoped", Entry: "Read", Files: map[string]string{"scoped.go": source}})
	if !first.Valid {
		t.Fatal(first.Diagnostics)
	}
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "nativeproof")
	if err != nil {
		t.Fatal(err)
	}
	functionAt := strings.Index(string(projected), "func Read")
	typeAt := strings.Index(string(projected), "type local struct")
	if functionAt < 0 || typeAt < functionAt {
		t.Fatalf("local type was hoisted or lost:\n%s", projected)
	}
	runNativeWithTest(t, map[string]string{"projected.go": string(projected)}, `func TestNative(t *testing.T) { if Read(9) != 9 { t.Fatal(Read(9)) } }`)
}
