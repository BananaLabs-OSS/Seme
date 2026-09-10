package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRejectsMissingAndUnexpectedArgumentsWithoutArtifacts(t *testing.T) {
	for _, args := range [][]string{nil, {"extra"}, {"-revision", "1"}} {
		var stderr bytes.Buffer
		if err := run(context.Background(), args, &stderr); err == nil {
			t.Fatalf("accepted %#v", args)
		}
	}
}

func TestStrictInputsRejectAliasesAndOutputContainment(t *testing.T) {
	d := t.TempDir()
	plain := filepath.Join(d, "plain")
	if err := os.WriteFile(plain, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := readStrict(plain); err != nil || !bytes.Equal(got, []byte("x")) {
		t.Fatalf("plain=%q err=%v", got, err)
	}
	alias := filepath.Join(d, "alias")
	if err := os.Symlink(plain, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := readStrict(alias); err == nil {
		t.Fatal("accepted symlink input")
	}
	if !inside(d, filepath.Join(d, "bundle")) || inside(d, filepath.Join(filepath.Dir(d), "bundle")) {
		t.Fatal("output containment")
	}
}
