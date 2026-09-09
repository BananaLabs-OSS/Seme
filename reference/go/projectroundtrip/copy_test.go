package projectroundtrip

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"seme.local/reference/projectsource"
)

func testPolicy() projectsource.Policy {
	return projectsource.Policy{
		TrackedExtensions: []string{".go"},
		IgnoredPrefixes:   []string{"ignored/"},
		VendoredPrefixes:  []string{"vendor/"},
		GeneratedHeader:   []byte("// Code generated "),
		MaxFiles:          16,
		MaxFileBytes:      1024,
		MaxTotalBytes:     4096,
	}
}

func put(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) (string, projectsource.Snapshot) {
	t.Helper()
	root := t.TempDir()
	put(t, root, "main.go", "package main\n")
	put(t, root, "generated.go", "// Code generated tool; DO NOT EDIT.\npackage main\n")
	put(t, root, "ignored/main_test.go", "package ignored\n")
	put(t, root, "vendor/example/lib.go", "package example\n")
	put(t, root, "NOTICE", "opaque bytes\x00\n")
	snapshot, err := projectsource.Discover(root, "example/project", projectsource.Toolchain{Language: "go", Toolchain: "go1.25", Profile: "bounded-v1", SemanticRevision: "go-bounded-v1"}, testPolicy())
	if err != nil {
		t.Fatal(err)
	}
	return root, snapshot
}

func TestCopyVerifiedPreservesEveryClassAndRediscovery(t *testing.T) {
	source, snapshot := fixture(t)
	dest := filepath.Join(t.TempDir(), "projected")
	if err := CopyVerified(source, dest, snapshot, testPolicy()); err != nil {
		t.Fatal(err)
	}
	if err := projectsource.Verify(dest, snapshot, testPolicy()); err != nil {
		t.Fatal(err)
	}
	for _, unit := range snapshot.Units {
		before, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(unit.Path)))
		if err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(unit.Path)))
		if err != nil {
			t.Fatal(err)
		}
		if string(before) != string(after) {
			t.Fatalf("bytes changed for %s", unit.Path)
		}
	}
}

func TestCopyVerifiedRejectsDriftWithoutDestination(t *testing.T) {
	source, snapshot := fixture(t)
	put(t, source, "main.go", "package changed\n")
	dest := filepath.Join(t.TempDir(), "projected")
	err := CopyVerified(source, dest, snapshot, testPolicy())
	if err == nil || !strings.Contains(err.Error(), "digest_drift") {
		t.Fatalf("err=%v", err)
	}
	if _, err := os.Lstat(dest); !os.IsNotExist(err) {
		t.Fatalf("partial destination exists: %v", err)
	}
}

func TestCopyVerifiedNeverOverwrites(t *testing.T) {
	source, snapshot := fixture(t)
	dest := filepath.Join(t.TempDir(), "projected")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	err := CopyVerified(source, dest, snapshot, testPolicy())
	if err == nil || !strings.Contains(err.Error(), "destination_exists") {
		t.Fatalf("err=%v", err)
	}
}

func TestCopyVerifiedNeverClobbersExistingFile(t *testing.T) {
	source, snapshot := fixture(t)
	dest := filepath.Join(t.TempDir(), "projected")
	const sentinel = "previous accepted output"
	if err := os.WriteFile(dest, []byte(sentinel), 0o600); err != nil {
		t.Fatal(err)
	}
	err := CopyVerified(source, dest, snapshot, testPolicy())
	if err == nil || !strings.Contains(err.Error(), "destination_exists") {
		t.Fatalf("err=%v", err)
	}
	data, readErr := os.ReadFile(dest)
	if readErr != nil || string(data) != sentinel {
		t.Fatalf("existing output changed: %q, %v", data, readErr)
	}
}
