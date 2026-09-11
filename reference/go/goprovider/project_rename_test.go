package goprovider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectRenamePublishesCrossPackageReferencesAtomically(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.test/live\n\ngo 1.25\n")
	write("model/value.go", "package model\n// Value remains in this comment.\nfunc Value(v int64) int64 { return v + 1 }\n")
	write("application/run.go", "package application\nimport \"example.test/live/model\"\nfunc Run(v int64) int64 { return model.Value(v) }\n")
	write("application/run_test.go", "package application\nimport \"testing\"\nfunc TestRun(t *testing.T) { if Run(4) != 5 { t.Fatal() } }\n")
	execution, err := os.ReadFile("../../../modules/execution/v36/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := NewIncrementalSession(execution)
	if err != nil {
		t.Fatal(err)
	}
	lifted := session.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/live", PackagePath: "example.test/live/application", Entry: "Run", Files: map[string]string{
		"model/value.go":     "package model\n// Value remains in this comment.\nfunc Value(v int64) int64 { return v + 1 }\n",
		"application/run.go": "package application\nimport \"example.test/live/model\"\nfunc Run(v int64) int64 { return model.Value(v) }\n",
	}})
	if !lifted.Valid {
		t.Fatalf("lift=%#v", lifted)
	}
	manifest, _, err := Ingest(IngestOptions{Project: root, ModuleG1: "../../../modules/provider/v1/module.g1", CanonicalSources: lifted.Sources})
	if err != nil {
		t.Fatal(err)
	}
	var target string
	for _, d := range manifest.Declarations {
		if d.Qualified == "example.test/live/model.Value" {
			target = d.ID
			for _, source := range lifted.Sources {
				if source.Name == "Value" && source.Document == "model/value.go" && source.ID != target {
					t.Fatalf("provider identity %s != semantic identity %s", target, source.ID)
				}
			}
			if len(d.Occurrences) < 2 {
				t.Fatalf("cross-package occurrences=%d", len(d.Occurrences))
			}
		}
	}
	if target == "" {
		t.Fatal("target missing")
	}
	destination := filepath.Join(t.TempDir(), "result")
	report, transcript, err := ProjectRename(root, destination, manifest, target, "Value", "Reading", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.ChangedFiles) != 2 || report.ResultRevision == report.BaseRevision || transcript == nil {
		t.Fatalf("report=%+v transcript=%q", report, transcript)
	}
	if len(report.IdentityBindings) != 1 || report.IdentityBindings[0].ID != target || report.IdentityBindings[0].Name != "Reading" || report.IdentityBindings[0].Document != "model/value.go" {
		t.Fatalf("continuity=%#v", report.IdentityBindings)
	}
	model, _ := os.ReadFile(filepath.Join(destination, "model/value.go"))
	app, _ := os.ReadFile(filepath.Join(destination, "application/run.go"))
	original, _ := os.ReadFile(filepath.Join(root, "model/value.go"))
	if !strings.Contains(string(model), "func Reading") || !strings.Contains(string(model), "// Value remains") || !strings.Contains(string(app), "model.Reading") || !strings.Contains(string(original), "func Value") {
		t.Fatal("incorrect atomic projection")
	}
	if _, _, err = ProjectRename(root, destination, manifest, target, "Value", "Again", false); err == nil {
		t.Fatal("overwrote destination")
	}
}

func TestProjectRenameRejectsStaleAndValidationFailureWithoutOutput(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/reject\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "value.go"), []byte("package reject\nfunc Value(v int64) int64{return v}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, _, err := Ingest(IngestOptions{Project: root, ModuleG1: "../../../modules/provider/v1/module.g1"})
	if err != nil {
		t.Fatal(err)
	}
	target := manifest.Declarations[0].ID
	if err = os.WriteFile(filepath.Join(root, "value.go"), []byte("package reject\nfunc Value(v int64) int64{return v+1}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "stale")
	if _, _, err = ProjectRename(root, destination, manifest, target, "Value", "Other", true); err == nil || !strings.Contains(err.Error(), "stale_native_revision") {
		t.Fatalf("stale err=%v", err)
	}
	if _, e := os.Lstat(destination); !os.IsNotExist(e) {
		t.Fatal("stale output published")
	}
}

func TestProjectRenameRejectsSymlinkedSourceWithoutOutput(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/symlink\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "value.go"), []byte("package symlink\nfunc Value(v int64) int64{return v}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, _, err := Ingest(IngestOptions{Project: root, ModuleG1: "../../../modules/provider/v1/module.g1"})
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err = os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "result")
	if _, _, err = ProjectRename(root, destination, manifest, manifest.Declarations[0].ID, "Value", "Reading", false); err == nil || !strings.Contains(err.Error(), "unsupported_source_entry") {
		t.Fatalf("symlink err=%v", err)
	}
	if _, statErr := os.Lstat(destination); !os.IsNotExist(statErr) {
		t.Fatal("symlinked output published")
	}
}
