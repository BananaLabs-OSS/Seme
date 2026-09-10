package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRejectsMissingFlagsAndUnsafePathsWithoutOutput(t *testing.T) {
	if err := run(context.Background(), nil, io.Discard); err == nil {
		t.Fatal("missing flags accepted")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readStrict(link); err == nil {
		t.Fatal("symlink input accepted")
	}
	if _, err := readStrict("relative"); err == nil {
		t.Fatal("relative input accepted")
	}
}

func TestReadProjectRejectsNonregularEntry(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "bad.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := readProject(dir); err == nil {
		t.Fatal("symlink project entry accepted")
	}
}
