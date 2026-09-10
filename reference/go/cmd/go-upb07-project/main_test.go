package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRejectsIncompleteArgumentsWithoutCreatingDestination(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "projected")
	err := run(context.Background(), []string{"-to", destination})
	if err == nil || !strings.Contains(err.Error(), "flag_missing") {
		t.Fatalf("error=%v", err)
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		t.Fatal("invalid invocation created destination")
	}
}

func TestContainmentUsesPathBoundaries(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "project")
	if !inside(root, filepath.Join(root, "child")) || inside(root, root+"-other") {
		t.Fatal("path containment boundary")
	}
}

func TestStrictReadersRejectRelativeAndSymlink(t *testing.T) {
	if _, err := readStrict("relative"); err == nil {
		t.Fatal("relative accepted")
	}
	d := t.TempDir()
	target := filepath.Join(d, "target")
	link := filepath.Join(d, "link")
	if err := os.WriteFile(target, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readStrict(link); err == nil {
		t.Fatal("symlink accepted")
	}
}
