package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRejectsMissingAndNonPlainInputsWithoutOutput(t *testing.T) {
	if err := run(nil); err == nil {
		t.Fatal("missing flags accepted")
	}
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain")
	if err := os.WriteFile(plain, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(plain, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readStrict(link); err == nil {
		t.Fatal("symlink input accepted")
	}
	if _, err := readStrict("relative"); err == nil {
		t.Fatal("relative input accepted")
	}
}
