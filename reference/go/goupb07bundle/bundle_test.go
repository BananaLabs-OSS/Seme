package goupb07bundle

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestManifestIsClosedOrderedAndCloneOwned(t *testing.T) {
	a := sampleArtifacts()
	b := []byte("opaque-resource\x00")
	s := sha256.Sum256(b)
	blobs := map[[32]byte][]byte{s: b}
	m := manifestFor(a, blobs)
	if !validManifest(m, a, blobs) {
		t.Fatal("valid manifest rejected")
	}
	tampered := clone(a)
	tampered.Durable[0] ^= 1
	if validManifest(m, tampered, blobs) {
		t.Fatal("artifact tamper accepted")
	}
	forged := cloneBlobs(blobs)
	forged[s][0] ^= 1
	if validManifest(m, a, forged) {
		t.Fatal("digest-forged blob accepted")
	}
	copyA := clone(a)
	copyB := cloneBlobs(blobs)
	a.ProjectV10[0] ^= 1
	blobs[s][0] ^= 1
	if bytes.Equal(copyA.ProjectV10, a.ProjectV10) || bytes.Equal(copyB[s], blobs[s]) {
		t.Fatal("clone aliases caller memory")
	}
	if r, err := Load(context.Background(), Input{Artifacts: copyA, Manifest: m, Blobs: copyB}); err == nil || len(r.Artifacts.ProjectV10) != 0 || len(r.Blobs) != 0 {
		t.Fatal("unauthenticated input returned partial authority")
	}
}

func TestReadDirectoryClosedLayout(t *testing.T) {
	a := sampleArtifacts()
	b := []byte("blob")
	s := sha256.Sum256(b)
	blobs := map[[32]byte][]byte{s: b}
	root := filepath.Join(t.TempDir(), "bundle")
	writeBundle(t, root, a, blobs)
	got, complete, gotBlobs, err := ReadDirectory(root)
	if err != nil || !bytes.Equal(got.ProjectV10, a.ProjectV10) || !bytes.Equal(complete, manifestFor(a, blobs)) || !bytes.Equal(gotBlobs[s], b) {
		t.Fatal("read", err)
	}
	if _, _, _, err = ReadDirectory(filepath.Join(root, ".")); err != nil {
		t.Fatal("clean absolute path rejected", err)
	}
}

func TestReadDirectoryRejectsMissingExtraSymlinkAndMalformedBlob(t *testing.T) {
	a := sampleArtifacts()
	makeBundle := func() string { root := filepath.Join(t.TempDir(), "bundle"); writeBundle(t, root, a, nil); return root }
	root := makeBundle()
	if err := os.Remove(filepath.Join(root, "project-v10.seme")); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ReadDirectory(root); err == nil {
		t.Fatal("missing accepted")
	}
	root = makeBundle()
	if err := os.WriteFile(filepath.Join(root, "extra"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ReadDirectory(root); err == nil {
		t.Fatal("extra accepted")
	}
	root = makeBundle()
	if err := os.Remove(filepath.Join(root, "durable-state-v1.seme")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("project-v10.seme", filepath.Join(root, "durable-state-v1.seme")); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ReadDirectory(root); err == nil {
		t.Fatal("symlink accepted")
	}
	root = makeBundle()
	if err := os.WriteFile(filepath.Join(root, "blobs", "not-a-digest"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ReadDirectory(root); err == nil {
		t.Fatal("malformed blob accepted")
	}
}

func sampleArtifacts() Artifacts {
	a := Artifacts{}
	for _, f := range files(a) {
		setArtifact(&a, f.name, []byte("artifact:"+f.name))
	}
	return a
}
func setArtifact(a *Artifacts, n string, b []byte) {
	switch n {
	case "construction-v36.g1":
		a.Construction = b
	case "execution-v36.seme":
		a.Execution = b
	case "project-base-v8.seme":
		a.ProjectBase = b
	case "inventory-v8.seme":
		a.Inventory = b
	case "package-detail-v4.seme":
		a.PackageDetail = b
	case "package-v4.seme":
		a.PackageV4 = b
	case "dependency-v1.seme":
		a.Dependency = b
	case "configuration-v3.seme":
		a.ConfigurationV3 = b
	case "project-v8.seme":
		a.ProjectV8 = b
	case "resource-v1.seme":
		a.Resource = b
	case "project-v9.seme":
		a.ProjectV9 = b
	case "durable-state-v1.seme":
		a.Durable = b
	case "source-presentation-v1.seme":
		a.Presentation = b
	case "project-v10.seme":
		a.ProjectV10 = b
	}
}
func manifestFor(a Artifacts, blobs map[[32]byte][]byte) []byte {
	r := []byte("seme-go-upb07-bundle-v1\n")
	for _, f := range files(a) {
		s := sha256.Sum256(f.data)
		r = append(r, []byte(f.name+" "+hex.EncodeToString(s[:])+"\n")...)
	}
	keys := make([][32]byte, 0, len(blobs))
	for k := range blobs {
		keys = append(keys, k)
	}
	sortSums(keys)
	for _, k := range keys {
		h := hex.EncodeToString(k[:])
		r = append(r, []byte("blobs/"+h+" "+h+"\n")...)
	}
	return r
}
func sortSums(x [][32]byte) {
	for i := 1; i < len(x); i++ {
		for j := i; j > 0 && bytes.Compare(x[j][:], x[j-1][:]) < 0; j-- {
			x[j], x[j-1] = x[j-1], x[j]
		}
	}
}
func writeBundle(t *testing.T, root string, a Artifacts, blobs map[[32]byte][]byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "blobs"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, f := range files(a) {
		if err := os.WriteFile(filepath.Join(root, f.name), f.data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for k, b := range blobs {
		if err := os.WriteFile(filepath.Join(root, "blobs", hex.EncodeToString(k[:])), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "COMPLETE.sha256"), manifestFor(a, blobs), 0600); err != nil {
		t.Fatal(err)
	}
}
