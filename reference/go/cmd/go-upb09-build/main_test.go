package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRejectsMissingAndUnexpectedArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"extra"}, {"-revision", "1"}} {
		var stderr bytes.Buffer
		if err := run(context.Background(), args, &stderr); err == nil {
			t.Fatalf("accepted %#v", args)
		}
	}
}

func TestStrictInputAndContainment(t *testing.T) {
	d := t.TempDir()
	plain := filepath.Join(d, "plain")
	if err := os.WriteFile(plain, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if b, err := readStrict(plain); err != nil || string(b) != "x" {
		t.Fatalf("%q %v", b, err)
	}
	alias := filepath.Join(d, "alias")
	if err := os.Symlink(plain, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := readStrict(alias); err == nil {
		t.Fatal("accepted alias")
	}
	if !inside(d, filepath.Join(d, "bundle")) {
		t.Fatal("containment")
	}
}

func TestUPB09InputsContainEffectsAndV12(t *testing.T) {
	o := options{effectsSelection: "/effects", controlledEffectsV1: "/controlled", projectV12: "/v12"}
	got := o.inputs()
	for _, want := range []string{o.effectsSelection, o.controlledEffectsV1, o.projectV12} {
		found := false
		for _, x := range got {
			found = found || x == want
		}
		if !found {
			t.Fatalf("missing %s", want)
		}
	}
}
