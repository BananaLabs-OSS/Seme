package goprojector_test

import (
	"bytes"
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
	projected, err := goprojector.Project([]byte(first.CanonicalG1), "sample")
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
