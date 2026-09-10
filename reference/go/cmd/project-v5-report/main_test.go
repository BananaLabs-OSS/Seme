package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFailureAndUnexpectedArgumentsWriteNothing(t *testing.T) {
	for _, args := range [][]string{nil, {"extra"}} {
		var out bytes.Buffer
		if err := run(args, &out); err == nil || out.Len() != 0 {
			t.Fatalf("err=%v out=%q", err, out.String())
		}
	}
}
func TestStrictReadRejectsRelativeSymlinkDirectory(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "x")
	if err := os.WriteFile(p, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(d, "link")
	if err := os.Symlink(p, link); err != nil {
		t.Fatal(err)
	}
	for _, x := range []string{"relative", link, d} {
		if _, err := readRegular(x); err == nil {
			t.Fatalf("accepted %s", x)
		}
	}
	if b, err := readRegular(p); err != nil || string(b) != "x" {
		t.Fatalf("b=%q err=%v", b, err)
	}
}
func TestMissingInputsNeverWritePartialJSON(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"--execution", "missing"}, &out); err == nil || out.Len() != 0 {
		t.Fatalf("err=%v out=%q", err, out.String())
	}
}
