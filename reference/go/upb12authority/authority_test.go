package upb12authority

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestAndAtomicPublication(t *testing.T) {
	files := fixtureFiles()
	manifest, err := Manifest(files)
	if err != nil || !strings.HasPrefix(string(manifest), Header+"\n") {
		t.Fatal(err, string(manifest))
	}
	root := t.TempDir()
	out := filepath.Join(root, "authority")
	if err = Publish(out, files); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(out)
	if err != nil || string(loaded[Required[0]]) != string(files[Required[0]]) {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(out, "COMPLETE.sha256")); err != nil || string(got) != string(manifest) {
		t.Fatal(err)
	}
	if err = Publish(out, files); err == nil {
		t.Fatal("overwrote publication")
	}
}
func fixtureFiles() map[string][]byte {
	blob := []byte("blob")
	sum := sha256.Sum256(blob)
	files := map[string][]byte{"blobs/" + hex.EncodeToString(sum[:]): blob}
	for _, name := range Required {
		files[name] = []byte(name)
	}
	return files
}
func TestRejectsNativeAndEscapingArtifacts(t *testing.T) {
	for _, name := range []string{"native.go", "native.js", "native.lua", "../escape.seme", "x\\y.seme", ""} {
		files := fixtureFiles()
		files[name] = []byte("x")
		if _, err := Manifest(files); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
}

func TestLoadRejectsTamperAndUndeclaredFiles(t *testing.T) {
	for _, kind := range []string{"tamper", "undeclared", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			out := filepath.Join(root, "authority")
			if err := Publish(out, fixtureFiles()); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "tamper":
				if err := os.WriteFile(filepath.Join(out, Required[0]), []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "undeclared":
				if err := os.WriteFile(filepath.Join(out, "extra.json"), []byte("{}"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(filepath.Join(out, Required[0]), filepath.Join(out, "alias.seme")); err != nil {
					t.Fatal(err)
				}
			}
			if value, err := Load(out); err == nil || value != nil {
				t.Fatalf("accepted %s", kind)
			}
		})
	}
}
