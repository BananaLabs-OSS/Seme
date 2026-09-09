package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishCreateOnlyNeverReplacesExistingArtifact(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "project.seme")
	if err := os.WriteFile(destination, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := publishCreateOnly(directory, destination, []byte("new")); err == nil {
		t.Fatal("replaced an existing artifact")
	}
	got, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(got, []byte("old")) {
		t.Fatalf("existing artifact changed: %q, %v", got, err)
	}
}

func TestReadProjectRejectsSymlinkAndNonregularInput(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("package fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "linked.go")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := readProject(root); err == nil {
		t.Fatal("accepted a symlink in the project tree")
	}
}

func TestInsideUsesPathComponents(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "project")
	if !inside(root, filepath.Join(root, "out.seme")) {
		t.Fatal("missed contained output")
	}
	if inside(root, filepath.Join(string(filepath.Separator), "project-other", "out.seme")) {
		t.Fatal("confused a path prefix with containment")
	}
}
