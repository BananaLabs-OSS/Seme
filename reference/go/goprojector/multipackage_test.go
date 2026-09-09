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
	if output, e := ProjectPackages([]byte(first.CanonicalG1), undeclared); e == nil || !strings.HasPrefix(e.Error(), "go_projection.undeclared_dependency:") || output != nil {
		t.Fatalf("undeclared call result=%#v error=%v", output, e)
	}
	inaccessible := clonePackageMetadata(first.Packages)
	for i := range inaccessible {
		if inaccessible[i].Name != "example.test/go-project-build-v1/model" {
			continue
		}
		for j := range inaccessible[i].Members {
			if inaccessible[i].Members[j].Name == "Normalize" {
				inaccessible[i].Members[j].Exported = false
			}
		}
		public := inaccessible[i].Functions[:0]
		for _, function := range inaccessible[i].Functions {
			if function.Name != "Normalize" {
				public = append(public, function)
			}
		}
		inaccessible[i].Functions = public
	}
	if output, e := ProjectPackages([]byte(first.CanonicalG1), inaccessible); e == nil || !strings.HasPrefix(e.Error(), "go_projection.inaccessible_member:") || output != nil {
		t.Fatalf("private call result=%#v error=%v", output, e)
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
	if err = os.MkdirAll(filepath.Join(dir, "vectors"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"requests.jsonl", "expected.jsonl"} {
		data, readErr := os.ReadFile(filepath.Join(root, "vectors", name))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if err = os.WriteFile(filepath.Join(dir, "vectors", name), data, 0644); err != nil {
			t.Fatal(err)
		}
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

func TestProjectPackagesOwnsAndEmitsPrivateHelpers(t *testing.T) {
	module, err := os.ReadFile("../../../modules/execution/v35/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"model/model.go":     "package model\nfunc normalize(v int64) int64 { return v + 1 }\nfunc Normalize(v int64) int64 { return normalize(v) }\n",
		"application/app.go": "package application\nimport \"example.test/private/model\"\nfunc Apply(v int64) int64 { return model.Normalize(v) }\n",
	}
	snapshot := goprovider.DocumentSnapshot{Revision: 1, ModulePath: "example.test/private", PackagePath: "example.test/private/application", Entry: "Apply", Files: files}
	session, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	first := session.Apply(snapshot)
	if !first.Valid {
		t.Fatal(first.Diagnostics)
	}
	var foundPrivate bool
	for _, p := range first.Packages {
		for _, member := range p.Members {
			if member.Name == "normalize" {
				foundPrivate = !member.Exported && member.Document == "model/model.go" && member.Line == 2 && member.Column > 0
			}
		}
		for _, public := range p.Functions {
			if public.Name == "normalize" {
				t.Fatal("private helper leaked into exported interfaces")
			}
		}
	}
	if !foundPrivate {
		t.Fatalf("private membership missing: %#v", first.Packages)
	}
	badMember := clonePackageMetadata(first.Packages)
	badMember[0].Members = append(badMember[0].Members, goprovider.PackageFunctionMetadata{ID: "ffffffffffffffffffffffffffffffff", Name: "forged", Result: badMember[0].Members[0].Result})
	if output, projectErr := ProjectPackages([]byte(first.CanonicalG1), badMember); projectErr == nil || !strings.HasPrefix(projectErr.Error(), "go_projection.function_not_program_member:") || output != nil {
		t.Fatalf("member forgery result=%#v error=%v", output, projectErr)
	}
	badExport := clonePackageMetadata(first.Packages)
	for i := range badExport {
		for j := range badExport[i].Members {
			if badExport[i].Members[j].Name == "normalize" {
				badExport[i].Members[j].Exported = true
			}
		}
	}
	if output, projectErr := ProjectPackages([]byte(first.CanonicalG1), badExport); projectErr == nil || !strings.HasPrefix(projectErr.Error(), "go_projection.export_missing:") || output != nil {
		t.Fatalf("export forgery result=%#v error=%v", output, projectErr)
	}
	projected, err := ProjectPackages([]byte(first.CanonicalG1), first.Packages)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(projected["example.test/private/model"]), "func normalize(") || !strings.Contains(string(projected["example.test/private/model"]), "return normalize(v)") {
		t.Fatal("private helper was not projected as a local declaration")
	}
	relift := map[string]string{}
	for packagePath, source := range projected {
		relative := strings.TrimPrefix(packagePath, snapshot.ModulePath+"/")
		relift[filepath.ToSlash(filepath.Join(relative, "projected.go"))] = string(source)
	}
	secondSession, err := goprovider.NewIncrementalSession(module)
	if err != nil {
		t.Fatal(err)
	}
	second := secondSession.Apply(goprovider.DocumentSnapshot{Revision: 1, ModulePath: snapshot.ModulePath, PackagePath: snapshot.PackagePath, Entry: snapshot.Entry, Files: relift})
	if !second.Valid || second.CanonicalG1 != first.CanonicalG1 {
		t.Fatalf("private-helper relift failed: %#v", second.Diagnostics)
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
		out[i].Members = clonePackageFunctions(p.Members)
		out[i].Functions = clonePackageFunctions(p.Functions)
	}
	return out
}

func clonePackageFunctions(input []goprovider.PackageFunctionMetadata) []goprovider.PackageFunctionMetadata {
	out := make([]goprovider.PackageFunctionMetadata, len(input))
	for i, f := range input {
		out[i] = f
		out[i].Parameters = append([]string(nil), f.Parameters...)
	}
	return out
}
