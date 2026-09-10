package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissingAndRelativeFlagsRejectWithoutOutput(t *testing.T) {
	var stderr bytes.Buffer
	if err := run(context.Background(), nil, &stderr); err == nil || !strings.Contains(err.Error(), "flag_missing") {
		t.Fatalf("err=%v", err)
	}
	out := filepath.Join(t.TempDir(), "bundle")
	args := completeArgs(t, t.TempDir(), out)
	args[1] = "relative"
	if err := run(context.Background(), args, &stderr); err == nil || !strings.Contains(err.Error(), "absolute_path") {
		t.Fatalf("err=%v", err)
	}
	assertAbsent(t, out)
}

func TestSymlinkAndEmptyProjectRejectWithoutOutput(t *testing.T) {
	parent := t.TempDir()
	real := filepath.Join(parent, "real")
	if err := os.Mkdir(real, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(parent, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(parent, "bundle")
	var stderr bytes.Buffer
	if err := run(context.Background(), completeArgs(t, link, out), &stderr); err == nil || !strings.Contains(err.Error(), "not_plain_directory") {
		t.Fatalf("err=%v", err)
	}
	assertAbsent(t, out)
	if err := run(context.Background(), completeArgs(t, real, out), &stderr); err == nil || !strings.Contains(err.Error(), "no_sources") {
		t.Fatalf("err=%v", err)
	}
	assertAbsent(t, out)
}

func TestExistingOutputPreservedBeforeInputWork(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "project")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(parent, "bundle")
	if err := os.Mkdir(out, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(out, "mine")
	if err := os.WriteFile(marker, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), completeArgs(t, project, out), &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "output_exists") {
		t.Fatalf("err=%v", err)
	}
	got, err := os.ReadFile(marker)
	if err != nil || string(got) != "keep" {
		t.Fatal("existing output changed")
	}
}

func TestRunPublishesConstructionAndCompletionManifest(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.Mkdir(project, 0700); err != nil {
		t.Fatal(err)
	}
	source := "package app\nfunc Apply(v int64) int64 { return v + 1 }\n"
	if err := os.WriteFile(filepath.Join(project, "main.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	repo, err := filepath.Abs("../../../../")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "bundle")
	args := []string{"-project", project, "-module", "example.test/cli", "-package", "example.test/cli", "-entry", "Apply", "-out", out, "-revision", "1", "-execution-g1", filepath.Join(repo, "modules/execution/v35/module.g1"), "-execution-contract", filepath.Join(repo, "modules/execution/v35/module.seme"), "-package-v1", filepath.Join(repo, "modules/package/v1/module.seme"), "-package-v2", filepath.Join(repo, "modules/package/v2/module.seme"), "-project-v1", filepath.Join(repo, "modules/project/v1/module.seme"), "-project-v2", filepath.Join(repo, "modules/project/v2/module.seme"), "-project-v3", filepath.Join(repo, "modules/project/v3/module.seme"), "-k0", filepath.Join(repo, "bootstrap/seme-k0-linux-amd64"), "-g1-compiler", filepath.Join(repo, "compiler/g1-compiler.k0")}
	if err = run(context.Background(), args, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"construction.g1", "project-v1.seme", "inventory-v2.seme", "package-v2.seme", "project-v3.seme", "COMPLETE.sha256"} {
		data, er := os.ReadFile(filepath.Join(out, name))
		if er != nil || len(data) == 0 {
			t.Fatalf("%s: %v", name, er)
		}
	}
}

func completeArgs(t *testing.T, project, out string) []string {
	t.Helper()
	dummy := filepath.Join(t.TempDir(), "input")
	if err := os.WriteFile(dummy, []byte("x"), 0700); err != nil {
		t.Fatal(err)
	}
	return []string{"-project", project, "-module", "example.test/x", "-package", "example.test/x", "-entry", "Apply", "-out", out, "-revision", "1", "-execution-g1", dummy, "-execution-contract", dummy, "-package-v1", dummy, "-package-v2", dummy, "-project-v1", dummy, "-project-v2", dummy, "-project-v3", dummy, "-k0", dummy, "-g1-compiler", dummy}
}
func assertAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("output exists: %v", err)
	}
}
