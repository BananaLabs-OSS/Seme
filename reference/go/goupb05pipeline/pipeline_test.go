package goupb05pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"seme.local/reference/goprojectpipeline"
	"seme.local/reference/goupb03pipeline"
	"seme.local/reference/goupb04pipeline"
)

func TestBuildRejectsUnauthenticatedContractsWithoutOutput(t *testing.T) {
	got, err := Build(t.Context(), Input{})
	if err == nil || got.ConfigurationV1 != nil || got.ProjectV7 != nil {
		t.Fatal("unauthenticated build returned output")
	}
}

func TestPublishIsCreateOnlyElevenArtifactsAndNoPartial(t *testing.T) {
	b := goprojectpipeline.Result{CanonicalG1: []byte("g1"), ProjectV1: []byte("p1"), InventoryV2: []byte("i2"), PackageV2: []byte("k2"), ProjectV3: []byte("p3")}
	r := Result{Base: goupb04pipeline.Result{Base: goupb03pipeline.Result{Base: b, DependencyV1: []byte("d1"), ProjectV4: []byte("p4")}, PackageV3: []byte("k3"), ProjectV5: []byte("p5")}, ConfigurationV2: []byte("c2"), ProjectV7: []byte("p7")}
	d := filepath.Join(t.TempDir(), "bundle")
	if err := Publish(d, r); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(d, "COMPLETE.sha256"))
	if err != nil {
		t.Fatal(err)
	}
	names := []string{"construction.g1", "project-v1.seme", "inventory-v2.seme", "package-v2.seme", "project-v3.seme", "dependency-v1.seme", "project-v4.seme", "package-v3.seme", "project-v5.seme", "configuration-v2.seme", "project-v7.seme"}
	for _, name := range names {
		if _, err = os.Stat(filepath.Join(d, name)); err != nil {
			t.Fatal(err)
		}
	}
	if lines := bytesCount(manifest, '\n'); lines != len(names)+1 {
		t.Fatalf("manifest lines %d", lines)
	}
	if err = Publish(d, r); err == nil {
		t.Fatal("overwrote destination")
	}
	bad := r
	bad.ProjectV7 = nil
	badPath := filepath.Join(filepath.Dir(d), "bad")
	if err = Publish(badPath, bad); err == nil {
		t.Fatal("accepted incomplete")
	}
	if _, err = os.Stat(badPath); !os.IsNotExist(err) {
		t.Fatal("left partial")
	}
}
func bytesCount(b []byte, x byte) int {
	n := 0
	for _, v := range b {
		if v == x {
			n++
		}
	}
	return n
}
