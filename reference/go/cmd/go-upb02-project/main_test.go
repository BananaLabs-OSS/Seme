package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTamperPublishesNothing(t *testing.T) {
	d := t.TempDir()
	root := filepath.Join(d, "source")
	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatal(err)
	}
	g := filepath.Join(d, "x.g1")
	a := filepath.Join(d, "x.seme")
	if err := os.WriteFile(g, []byte("bad"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a, []byte("tampered"), 0644); err != nil {
		t.Fatal(err)
	}
	to := filepath.Join(d, "out")
	if run([]string{"--root", root, "--construction", g, "--package-v2", a, "--module", "example.test/tool", "--to", to}) == nil {
		t.Fatal("accepted tamper")
	}
	if _, e := os.Lstat(to); !os.IsNotExist(e) {
		t.Fatalf("partial output: %v", e)
	}
}
func TestDestinationIsCreateOnly(t *testing.T) {
	d := t.TempDir()
	root := filepath.Join(d, "source")
	to := filepath.Join(d, "out")
	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(to, 0755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(to, "sentinel")
	if err := os.WriteFile(sentinel, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if run([]string{"--root", root, "--construction", filepath.Join(d, "missing"), "--package-v2", filepath.Join(d, "missing2"), "--module", "example.test/tool", "--to", to}) == nil {
		t.Fatal("accepted existing destination")
	}
	b, _ := os.ReadFile(sentinel)
	if string(b) != "keep" {
		t.Fatal("existing destination changed")
	}
}
func TestReadStrictRejectsSymlinkAndOversize(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "target")
	link := filepath.Join(d, "link")
	if err := os.WriteFile(target, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, e := readStrict(link); e == nil {
		t.Fatal("accepted symlink")
	}
	large := filepath.Join(d, "large")
	f, e := os.Create(large)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.Truncate((64 << 20) + 1); e != nil {
		t.Fatal(e)
	}
	f.Close()
	if _, e = readStrict(large); e == nil {
		t.Fatal("accepted oversized input")
	}
}
func TestUnexpectedArgumentsRejectSilently(t *testing.T) {
	if run([]string{"extra"}) == nil {
		t.Fatal("accepted positional argument")
	}
}
