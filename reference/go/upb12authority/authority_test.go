package upb12authority

import (
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
	if got, err := os.ReadFile(filepath.Join(out, "COMPLETE.sha256")); err != nil || string(got) != string(manifest) {
		t.Fatal(err)
	}
	if err = Publish(out, files); err == nil {
		t.Fatal("overwrote publication")
	}
}
func fixtureFiles() map[string][]byte {
	files := map[string][]byte{"blobs/" + strings.Repeat("0", 64): []byte("blob")}
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
