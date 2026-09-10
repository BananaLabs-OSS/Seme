package goupb05bundle

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestManifestAuthenticatesEveryExactArtifact(t *testing.T) {
	a := Artifacts{[]byte("g"), []byte("p1"), []byte("i2"), []byte("k2"), []byte("p3"), []byte("d1"), []byte("p4"), []byte("k3"), []byte("p5"), []byte("c2"), []byte("p7")}
	m := manifest(a)
	if !validManifest(m, a) {
		t.Fatal("valid manifest rejected")
	}
	bad := a
	bad.ProjectV7 = []byte("different")
	if validManifest(m, bad) {
		t.Fatal("mix-and-match accepted")
	}
	extra := append(append([]byte(nil), m...), byte('x'))
	if validManifest(extra, a) {
		t.Fatal("extra manifest content accepted")
	}
}
func TestLoadRejectsZeroContractsWithoutOutput(t *testing.T) {
	got, err := Load(t.Context(), Input{})
	if err == nil || got.ConfigurationV1 != nil || got.ProjectV6 != nil {
		t.Fatal("returned partial output")
	}
}
func manifest(a Artifacts) []byte {
	files := []struct {
		name string
		data []byte
	}{{"construction.g1", a.Construction}, {"project-v1.seme", a.ProjectV1}, {"inventory-v2.seme", a.InventoryV2}, {"package-v2.seme", a.PackageV2}, {"project-v3.seme", a.ProjectV3}, {"dependency-v1.seme", a.DependencyV1}, {"project-v4.seme", a.ProjectV4}, {"package-v3.seme", a.PackageV3}, {"project-v5.seme", a.ProjectV5}, {"configuration-v2.seme", a.ConfigurationV2}, {"project-v7.seme", a.ProjectV7}}
	out := []byte("seme-go-upb05-bundle-v1\n")
	for _, f := range files {
		s := sha256.Sum256(f.data)
		out = append(out, []byte(f.name+" "+hex.EncodeToString(s[:])+"\n")...)
	}
	return out
}
