package goprovider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectRenameBundlePublishesCompleteReingestedEvidence(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.test/bundle\n\ngo 1.25\n")
	writeFile(t, root, "model/value.go", "package model\nfunc Value(v int64) int64 { return v + 1 }\n")
	writeFile(t, root, "app/run.go", "package app\nimport \"example.test/bundle/model\"\nfunc Run(v int64) int64 { return model.Value(v) }\n")
	writeFile(t, root, "app/run_test.go", "package app\nimport \"testing\"\nfunc TestRun(t *testing.T) { if Run(1) != 2 { t.Fatal() } }\n")
	execution, err := os.ReadFile("../../../modules/execution/v36/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, _ := NewIncrementalSession(execution)
	lifted := session.Apply(DocumentSnapshot{Revision: 1, ModulePath: "example.test/bundle", PackagePath: "example.test/bundle/app", Entry: "Run", Files: map[string]string{
		"model/value.go": "package model\nfunc Value(v int64) int64 { return v + 1 }\n",
		"app/run.go":     "package app\nimport \"example.test/bundle/model\"\nfunc Run(v int64) int64 { return model.Value(v) }\n",
	}})
	if !lifted.Valid {
		t.Fatalf("lift=%#v", lifted)
	}
	provider := "../../../modules/provider/v1/module.g1"
	prior, _, err := Ingest(IngestOptions{Project: root, ModuleG1: provider, CanonicalSources: lifted.Sources})
	if err != nil {
		t.Fatal(err)
	}
	var target string
	for _, declaration := range prior.Declarations {
		if declaration.Qualified == "example.test/bundle/model.Value" {
			target = declaration.ID
		}
	}
	destination := filepath.Join(t.TempDir(), "result")
	bundle, err := ProjectRenameBundle(root, destination, provider, prior, target, "Value", "Reading")
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Prior.Revision == bundle.Result.Revision || bundle.Report.ResultRevision != bundle.Result.Revision || len(bundle.NativeValidationTranscript) == 0 {
		t.Fatalf("bundle=%#v", bundle)
	}
	foundIdentity := false
	for _, declaration := range bundle.Result.Declarations {
		if declaration.Name == "Reading" && declaration.ID == target {
			foundIdentity = true
		}
	}
	if !foundIdentity {
		t.Fatal("re-ingestion lost renamed declaration identity")
	}
	for _, name := range []string{"prior-provider.json", "result-provider.json", "prior-provider.g1", "result-provider.g1", "projection-report.json", "native-validation.txt", "COMPLETE.sha256"} {
		if info, statErr := os.Stat(filepath.Join(destination, reconciliationDirectory, name)); statErr != nil || info.Size() == 0 {
			t.Fatalf("missing %s: %v", name, statErr)
		}
	}
	updated, _ := os.ReadFile(filepath.Join(destination, "app/run.go"))
	if !strings.Contains(string(updated), "model.Reading") {
		t.Fatal("project was not renamed")
	}
	reopened, err := ReadReconciliationBundle(destination, provider)
	if err != nil || reopened.Result.Revision != bundle.Result.Revision || reopened.Report.IdentityBindings[0].ID != target {
		t.Fatalf("reopen=%#v err=%v", reopened, err)
	}
	manifestPath := filepath.Join(destination, reconciliationDirectory, "COMPLETE.sha256")
	complete, _ := os.ReadFile(manifestPath)
	if err = os.WriteFile(manifestPath, append(complete, 'x'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = ReadReconciliationBundle(destination, provider); err == nil || !strings.Contains(err.Error(), "reconciliation_manifest") {
		t.Fatalf("tampered completeness accepted: %v", err)
	}
}

func writeFile(t *testing.T, root, name, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
