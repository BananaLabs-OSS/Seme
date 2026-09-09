package goprojector

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"seme.local/reference/goprovider"
)

func TestProjectPackagesPreservesOwnershipCallsAndRelift(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v35/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	root := "../../../fixtures/go-project-build-v1"
	for _, name := range []string{"model/value.go", "policy/policy.go", "application/application.go"} {
		b, e := os.ReadFile(filepath.Join(root, name))
		if e != nil {
			t.Fatal(e)
		}
		files[name] = string(b)
	}
	snapshot := goprovider.DocumentSnapshot{Revision: 1, ModulePath: "example.test/go-project-build-v1", PackagePath: "example.test/go-project-build-v1/application", Entry: "Apply", Files: files}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(snapshot)
	if !first.Valid {
		t.Fatal(first.Diagnostics)
	}
	projected, err := ProjectPackages([]byte(first.CanonicalG1), first.Packages)
	if err != nil {
		t.Fatal(err)
	}
	undeclared := clonePackageMetadata(first.Packages)
	for i := range undeclared {
		if undeclared[i].Name == snapshot.PackagePath {
			undeclared[i].Dependencies = nil
		}
	}
	if _, e := ProjectPackages([]byte(first.CanonicalG1), undeclared); e == nil || !strings.Contains(e.Error(), "undeclared_dependency") {
		t.Fatalf("accepted undeclared cross-package call: %v", e)
	}
	if len(projected) != 3 {
		t.Fatalf("files=%d", len(projected))
	}
	if !strings.Contains(string(projected["example.test/go-project-build-v1/application"]), "policy.Apply(value)") {
		t.Fatal("cross-package application call not qualified")
	}
	if !strings.Contains(string(projected["example.test/go-project-build-v1/policy"]), "model.Normalize(value)") {
		t.Fatal("cross-package policy call not qualified")
	}
	if strings.Count(string(projected["example.test/go-project-build-v1/application"]), "func Apply(") != 1 || strings.Count(string(projected["example.test/go-project-build-v1/policy"]), "func Apply(") != 1 {
		t.Fatal("duplicate function names lost package ownership")
	}
	dir := t.TempDir()
	if err = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.test/go-project-build-v1\n\ngo 1.25\n"), 0644); err != nil {
		t.Fatal(err)
	}
	reliftFiles := map[string]string{}
	for packagePath, source := range projected {
		relative := strings.TrimPrefix(packagePath, "example.test/go-project-build-v1/")
		name := filepath.ToSlash(filepath.Join(relative, "projected.go"))
		if err = os.MkdirAll(filepath.Join(dir, relative), 0755); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name), source, 0644); err != nil {
			t.Fatal(err)
		}
		reliftFiles[name] = string(source)
	}
	testSource, err := os.ReadFile(filepath.Join(root, "application/application_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "application/application_test.go"), testSource, 0644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "./...")
	command.Dir = dir
	command.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "go-cache"))
	if output, e := command.CombinedOutput(); e != nil {
		t.Fatalf("projected native go test: %v\n%s", e, output)
	}
	secondSession, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	second := secondSession.Apply(goprovider.DocumentSnapshot{Revision: 1, ModulePath: snapshot.ModulePath, PackagePath: snapshot.PackagePath, Entry: snapshot.Entry, Files: reliftFiles})
	if !second.Valid {
		t.Fatal(second.Diagnostics)
	}
	if second.CanonicalG1 != first.CanonicalG1 {
		t.Fatal("multi-package projection did not relift byte-identically")
	}
}

func TestProjectPackagesRejectsOwnershipForgery(t *testing.T) {
	module, _ := os.ReadFile("../../../modules/execution/v35/module.g1")
	session, _ := goprovider.NewIncrementalSession(module)
	result := session.Apply(goprovider.DocumentSnapshot{Revision: 1, ModulePath: "x", PackagePath: "x", Entry: "Apply", Files: map[string]string{"x.go": "package x\nfunc Apply(v int64) int64 { return v }\n"}})
	if !result.Valid {
		t.Fatal(result.Diagnostics)
	}
	badName := clonePackageMetadata(result.Packages)
	badName[0].Functions[0].Name = "Forged"
	badParameter := clonePackageMetadata(result.Packages)
	badParameter[0].Functions[0].Parameters = []string{"forged"}
	badResult := clonePackageMetadata(result.Packages)
	badResult[0].Functions[0].Result = "forged"
	tests := map[string][]goprovider.PackageMetadata{"missing": {}, "duplicate-package": {result.Packages[0], result.Packages[0]}, "duplicate-owner": {result.Packages[0], {Name: "other", Functions: result.Packages[0].Functions}}, "name": badName, "parameter": badParameter, "result": badResult, "self-dependency": {{Name: "x", Dependencies: []string{"x"}, Functions: result.Packages[0].Functions}}, "unknown-dependency": {{Name: "x", Dependencies: []string{"missing"}, Functions: result.Packages[0].Functions}}, "duplicate-dependency": {{Name: "x", Dependencies: []string{"other", "other"}, Functions: result.Packages[0].Functions}, {Name: "other"}}}
	for name, metadata := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := ProjectPackages([]byte(result.CanonicalG1), metadata); err == nil {
				t.Fatal("accepted forged ownership")
			}
		})
	}
}

func clonePackageMetadata(input []goprovider.PackageMetadata) []goprovider.PackageMetadata {
	out := make([]goprovider.PackageMetadata, len(input))
	for i, p := range input {
		out[i] = p
		out[i].Dependencies = append([]string(nil), p.Dependencies...)
		out[i].Functions = make([]goprovider.PackageFunctionMetadata, len(p.Functions))
		for j, f := range p.Functions {
			out[i].Functions[j] = f
			out[i].Functions[j].Parameters = append([]string(nil), f.Parameters...)
		}
	}
	return out
}
