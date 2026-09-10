package goupb08bundle

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func TestManifestRejectsArtifactAndBlobTamper(t *testing.T) {
	a := fakeArtifacts()
	blob := []byte("blob")
	sum := sha256.Sum256(blob)
	blobs := map[[32]byte][]byte{sum: blob}
	m, ok := manifest("seme-go-upb08-bundle-v1", files(a), blobs)
	if !ok || !validManifest(m, a, blobs) {
		t.Fatal("valid rejected")
	}
	tampered := clone(a)
	tampered.Transport = []byte("other")
	if validManifest(m, tampered, blobs) {
		t.Fatal("artifact tamper accepted")
	}
	badBlobs := cloneBlobs(blobs)
	badBlobs[sum] = []byte("other")
	if validManifest(m, a, badBlobs) {
		t.Fatal("blob tamper accepted")
	}
}
func TestReadDirectoryClosedWorld(t *testing.T) {
	a := fakeArtifacts()
	root := filepath.Join(t.TempDir(), "bundle")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "blobs"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, f := range files(a) {
		if err := os.WriteFile(filepath.Join(root, f.name), f.data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	m, _ := manifest("seme-go-upb08-bundle-v1", files(a), nil)
	if err := os.WriteFile(filepath.Join(root, "COMPLETE.sha256"), m, 0600); err != nil {
		t.Fatal(err)
	}
	got, complete, _, err := ReadDirectory(root)
	if err != nil || !bytes.Equal(got.ProjectV11, a.ProjectV11) || !bytes.Equal(complete, m) {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "extra"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err = ReadDirectory(root); err == nil {
		t.Fatal("extra accepted")
	}
}
func fakeArtifacts() Artifacts {
	a := Artifacts{}
	for i, f := range files(a) {
		setArtifact(&a, f.name, []byte{byte(i + 1)})
	}
	return a
}
func setArtifact(a *Artifacts, n string, b []byte) {
	switch n {
	case "construction-v36.g1":
		a.Base.Construction = b
	case "execution-v36.seme":
		a.Base.Execution = b
	case "project-base-v8.seme":
		a.Base.ProjectBase = b
	case "inventory-v8.seme":
		a.Base.Inventory = b
	case "package-detail-v4.seme":
		a.Base.PackageDetail = b
	case "package-v4.seme":
		a.Base.PackageV4 = b
	case "dependency-v1.seme":
		a.Base.Dependency = b
	case "configuration-v3.seme":
		a.Base.ConfigurationV3 = b
	case "project-v8.seme":
		a.Base.ProjectV8 = b
	case "resource-v1.seme":
		a.Base.Resource = b
	case "project-v9.seme":
		a.Base.ProjectV9 = b
	case "durable-state-v1.seme":
		a.Base.Durable = b
	case "source-presentation-v1.seme":
		a.Base.Presentation = b
	case "project-v10.seme":
		a.Base.ProjectV10 = b
	case "ordered-transport-v1.seme":
		a.Transport = b
	case "project-v11.seme":
		a.ProjectV11 = b
	}
}
