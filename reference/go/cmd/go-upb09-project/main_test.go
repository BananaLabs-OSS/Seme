package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRejectsIncompleteWithoutProjection(t *testing.T) {
	to := filepath.Join(t.TempDir(), "projected")
	if err := run(t.Context(), []string{"-to", to, "-module", "example.test/x", "-go", "/missing"}); err == nil {
		t.Fatal("projected unauthenticated")
	}
	if _, err := os.Lstat(to); !os.IsNotExist(err) {
		t.Fatal("partial projection")
	}
}
func TestDestinationAndPathRules(t *testing.T) {
	d := t.TempDir()
	if !cleanAbsolute(d) || !plainDirectory(d) || !safeRelative("a/b.go") || safeRelative("../b") || safeRelative("/b") {
		t.Fatal("path rules")
	}
	to := filepath.Join(d, "exists")
	if err := os.Mkdir(to, 0700); err != nil {
		t.Fatal(err)
	}
	if err := run(t.Context(), []string{"-to", to, "-module", "x", "-go", "/missing"}); err == nil {
		t.Fatal("accepted existing destination")
	}
}
func TestSymlinkParentRejected(t *testing.T) {
	d := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(d, alias); err != nil {
		t.Fatal(err)
	}
	if plainDirectory(alias) {
		t.Fatal("accepted symlink")
	}
}
