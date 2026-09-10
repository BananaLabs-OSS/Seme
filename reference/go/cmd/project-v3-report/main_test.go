package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"seme.local/reference/projectv3report"
)

func TestRunFailureWritesNoPartialStdout(t *testing.T) {
	var out bytes.Buffer
	if err := run(nil, &out); err == nil || out.Len() != 0 {
		t.Fatalf("err=%v stdout=%q", err, out.String())
	}
	dir := t.TempDir()
	p := filepath.Join(dir, "x")
	if err := os.WriteFile(p, []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{}
	for _, n := range []string{"execution", "package-v1", "package-v2", "project-v2-contract", "project-v3-contract", "project", "inventory", "package-graph", "composed"} {
		args = append(args, "--"+n, p)
	}
	if err := run(args, &out); err == nil || out.Len() != 0 {
		t.Fatalf("err=%v stdout=%q", err, out.String())
	}
}
func TestStableJSONEncoding(t *testing.T) {
	report := projectv3report.Report{Root: "root", Packages: []projectv3report.Package{{Identity: "root", ID: "01", Root: true}}}
	a, err := encode(report)
	if err != nil {
		t.Fatal(err)
	}
	b, err := encode(report)
	if err != nil || !bytes.Equal(a, b) {
		t.Fatal("unstable")
	}
	if len(a) == 0 || a[len(a)-1] != '\n' {
		t.Fatalf("not newline terminated: %q", a)
	}
}
func TestReadRegularRejectsSymlinkAndDirectory(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "target")
	if err := os.WriteFile(target, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(d, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegular(link); err == nil {
		t.Fatal("symlink accepted")
	}
	if _, err := readRegular(d); err == nil {
		t.Fatal("directory accepted")
	}
	b, err := readRegular(target)
	if err != nil || string(b) != "x" {
		t.Fatalf("b=%q err=%v", b, err)
	}
}
