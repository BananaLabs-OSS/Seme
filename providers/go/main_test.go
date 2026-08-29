package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRenameRoundTrip(t *testing.T) {
	fixture := filepath.Join("testdata", "ordinary")
	root := t.TempDir()
	entries, err := os.ReadDir(fixture)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(fixture, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, entry.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	before, err := ingest(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	var target Entity
	for _, entity := range before.Entities {
		if entity.Name == "Greeting" {
			target = entity
		}
	}
	if target.ID == "" || len(target.Occurrences) != 3 {
		t.Fatalf("unexpected target: %#v", target)
	}
	patch := Patch{Contract: "seme.patch/v1", BaseRevision: before.Revision, Renames: []Rename{{Entity: target.ID, ExpectedName: "Greeting", NewName: "Welcome"}}}
	if err := verifyNativeDigests(before); err != nil {
		t.Fatal(err)
	}
	if err := applyPatch(before, patch); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native tests: %v\n%s", err, output)
	}
	after, err := ingest(root, &before)
	if err != nil {
		t.Fatal(err)
	}
	afterByID := map[string]Entity{}
	for _, entity := range after.Entities {
		afterByID[entity.ID] = entity
	}
	for _, entity := range before.Entities {
		if _, ok := afterByID[entity.ID]; !ok {
			t.Fatalf("unaffected identity was not recovered: %#v", entity)
		}
	}
	var recovered Entity
	for _, entity := range after.Entities {
		if entity.ID == target.ID {
			recovered = entity
		}
	}
	if recovered.Name != "Welcome" {
		t.Fatalf("identity not recovered: %#v", recovered)
	}
	data, err := os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "package ordinary\n\n// This comment and spacing must survive a semantic rename.\nfunc Welcome(name string) string {\n\treturn \"Hello, \" + name\n}\n\nfunc Message() string {\n\treturn Welcome(\"Seme\")\n}\n" {
		t.Fatalf("unrelated bytes changed:\n%s", data)
	}
}

func TestRejectsNativeChangeAfterIngestion(t *testing.T) {
	root := copyFixture(t)
	program, err := ingest(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "main.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, []byte("// native edit\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyNativeDigests(program); err == nil {
		t.Fatal("changed native source was accepted")
	}
}

func copyFixture(t *testing.T) string {
	t.Helper()
	fixture := filepath.Join("testdata", "ordinary")
	root := t.TempDir()
	entries, err := os.ReadDir(fixture)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(fixture, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, entry.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
