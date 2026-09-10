package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/dependencyemitter"
	"seme.local/reference/dependencyresolution"
)

func TestRunSuccessStableJSON(t *testing.T) {
	contractPath := "../../../../modules/dependency/v1/module.seme"
	contractBytes, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := contractcatalog.ResolveDependencyContract(contractBytes)
	if err != nil {
		t.Fatal(err)
	}
	closure := dependencyresolution.Closure{Requirements: []dependencyresolution.Requirement{{Identity: "example.test/a", Requirement: "v1", Kind: dependencyresolution.External}}, Entries: []dependencyresolution.Entry{{Identity: "example.test/a", Ecosystem: "registry", Version: "v1", Integrity: "sha256:abc", IntegrityAlgorithm: "sha256", Source: "registry/a", SourceKind: "registry", Digest: "abc", Kind: dependencyresolution.External}}}
	instance, err := dependencyemitter.Emit(contract, closure)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "instance.seme")
	if err = os.WriteFile(p, instance, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--contract", contractPath, "--instance", p}
	var a, b bytes.Buffer
	if err = run(args, &a); err != nil {
		t.Fatal(err)
	}
	if err = run(args, &b); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) || !bytes.Contains(a.Bytes(), []byte(`"source_digest":"abc"`)) {
		t.Fatalf("unstable/incomplete: %s", a.Bytes())
	}
}

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
	if err := run([]string{"--contract", bad, "--instance", bad}, &out); err == nil || out.Len() != 0 {
		t.Fatalf("err=%v out=%q", err, out.String())
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
