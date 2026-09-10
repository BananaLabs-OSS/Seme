package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"seme.local/reference/projectv4report"
)

func TestFailureWritesNoPartialOutput(t *testing.T) {
	var out bytes.Buffer
	if err := run(nil, &out); err == nil || out.Len() != 0 {
		t.Fatalf("err=%v out=%q", err, out.String())
	}
	d := t.TempDir()
	bad := filepath.Join(d, "bad")
	if err := os.WriteFile(bad, []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{}
	for _, n := range []string{"execution", "package-v1", "package-v2", "dependency-contract", "project-v2-contract", "project-v3-contract", "project-v4-contract", "project", "inventory", "package-graph", "project-v3", "dependency", "composed"} {
		args = append(args, "--"+n, bad)
	}
	if err := run(args, &out); err == nil || out.Len() != 0 {
		t.Fatalf("err=%v out=%q", err, out.String())
	}
}
func TestStableJSONEncoding(t *testing.T) {
	r := projectv4report.Report{ContractModule: "e000", ContractRevision: "e004", Snapshot: projectv4report.Snapshot{ID: "snapshot"}}
	a, err := encode(r)
	if err != nil {
		t.Fatal(err)
	}
	b, err := encode(r)
	if err != nil || !bytes.Equal(a, b) {
		t.Fatal("unstable encoding")
	}
	if len(a) == 0 || a[len(a)-1] != '\n' {
		t.Fatalf("not newline terminated: %q", a)
	}
}
func TestReadRegularRejectsSymlinkAndDirectory(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "x")
	if err := os.WriteFile(p, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(d, "link")
	if err := os.Symlink(p, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegular(link); err == nil {
		t.Fatal("symlink accepted")
	}
	if _, err := readRegular(d); err == nil {
		t.Fatal("directory accepted")
	}
	if b, err := readRegular(p); err != nil || string(b) != "x" {
		t.Fatalf("b=%q err=%v", b, err)
	}
}
func TestUnexpectedArgumentsWriteNothing(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"extra"}, &out); err == nil || out.Len() != 0 {
		t.Fatalf("err=%v out=%q", err, out.String())
	}
}
