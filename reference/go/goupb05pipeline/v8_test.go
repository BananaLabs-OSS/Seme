package goupb05pipeline

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildV8RejectsUnauthenticatedInputWithoutOutput(t *testing.T) {
	got, err := BuildV8(t.Context(), V8Input{})
	if err == nil || len(got.ProjectV8) != 0 || len(got.Execution) != 0 {
		t.Fatal("unauthenticated input accepted or returned partial output")
	}
}

func TestPublishV8IsDeterministicCreateOnlyAndCompleteLast(t *testing.T) {
	r := V8Result{
		Construction: []byte("g1"), Execution: []byte("execution"),
		ProjectBase: []byte("project"), Inventory: []byte("inventory"),
		PackageDetail: []byte("detail"), PackageV4: []byte("package"),
		Dependency: []byte("dependency"), ConfigurationV3: []byte("configuration"),
		ProjectV8: []byte("complete"),
	}
	first := filepath.Join(t.TempDir(), "first")
	second := filepath.Join(t.TempDir(), "second")
	if err := PublishV8(first, r); err != nil {
		t.Fatal(err)
	}
	if err := PublishV8(second, r); err != nil {
		t.Fatal(err)
	}
	a, err := os.ReadFile(filepath.Join(first, "COMPLETE.sha256"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(second, "COMPLETE.sha256"))
	if err != nil || string(a) != string(b) {
		t.Fatalf("nondeterministic manifest: %v", err)
	}
	for _, file := range v8Files(r) {
		if _, err = os.Stat(filepath.Join(first, file.name)); err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(a, []byte(file.name+" ")) {
			t.Fatalf("manifest omitted %s", file.name)
		}
	}
	if err = PublishV8(first, r); err == nil {
		t.Fatal("existing bundle overwritten")
	}
	bad := r
	bad.PackageV4 = nil
	partial := filepath.Join(filepath.Dir(first), "partial")
	if err = PublishV8(partial, bad); err == nil {
		t.Fatal("empty component accepted")
	}
	if _, err = os.Stat(partial); !os.IsNotExist(err) {
		t.Fatal("partial bundle remained")
	}
}
