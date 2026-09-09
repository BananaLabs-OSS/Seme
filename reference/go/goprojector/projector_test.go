package goprojector_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"seme.local/reference/goprojector"
	"seme.local/reference/goprovider"
)

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

func runNative(t *testing.T, files map[string]string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.test/go-uab-call\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	testSource := "package nativeproof\nimport \"testing\"\nfunc TestNative(t *testing.T) { if got := Render(\"a\", \"b\"); got != \"[[a][b]]\" { t.Fatal(got) } }\n"
	if err := os.WriteFile(filepath.Join(dir, "projected_test.go"), []byte(testSource), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "-buildvcs=false", "./...")
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("native projected Go failed: %v\n%s", err, output)
	}
}

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
	unsupported := strings.Replace(result.CanonicalG1, "00000000000000000000000000009014 1 3", "000000000000000000000000000090a0 1 3", 1)
	if _, err := goprojector.Project([]byte(unsupported), "reject"); err == nil || !strings.Contains(err.Error(), "go_projection.unsupported_expression") {
		t.Fatalf("unsupported semantic expression accepted: %v", err)
	}
}
