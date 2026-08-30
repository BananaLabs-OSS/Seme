package goprovider

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIngestModuleWithoutRootPackage(t *testing.T) {
	project := t.TempDir()
	writeTestFile(t, filepath.Join(project, "go.mod"), "module example.com/rootless\n\ngo 1.26\n")
	writeTestFile(t, filepath.Join(project, "alpha", "alpha.go"), "package alpha\nfunc A() {}\n")
	writeTestFile(t, filepath.Join(project, "cmd", "tool", "main.go"), "package main\nfunc main() {}\n")
	module := filepath.Join(project, "provider.g1")
	writeTestFile(t, module, "# module\nve 1\nmo 00000000000000000000000000007000\nrv 00000000000000000000000000007001\npc 0\nec 0\n")

	manifest, _, err := Ingest(IngestOptions{Project: project, ModuleG1: module})
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Declarations) != 2 {
		t.Fatalf("declarations = %d, want 2", len(manifest.Declarations))
	}
	if manifest.Declarations[0].Qualified != "example.com/rootless/alpha.A" && manifest.Declarations[1].Qualified != "example.com/rootless/alpha.A" {
		t.Fatal("nested package declaration missing")
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
