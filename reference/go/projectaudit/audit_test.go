package projectaudit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanClassifiesProjectEcosystems(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "go-app", "go.mod"), "module example.com/go-app")
	write(t, filepath.Join(root, "web", "package.json"), "{}")
	write(t, filepath.Join(root, "mixed", "go.mod"), "module example.com/mixed")
	write(t, filepath.Join(root, "mixed", "native", "Cargo.toml"), "[package]")
	if err := os.Mkdir(filepath.Join(root, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	report, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Total != 4 || report.Summary.GoProviderReady != 1 || report.Summary.ProviderRequired != 2 || report.Summary.MetadataOnly != 1 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	if report.Projects[1].Name != "mixed" || report.Projects[1].Provider != "seme.go-provider.v1+additional-provider-required" {
		t.Fatalf("mixed project not classified: %+v", report.Projects[1])
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
