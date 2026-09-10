package goupb04pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprojectpipeline"
	"seme.local/reference/goupb03pipeline"
)

func TestProjectRequiresAuthenticatedCompleteOwnership(t *testing.T) {
	if out, err := Project(Result{}, contractcatalog.ProjectContractSetV5{}); err == nil || out != nil {
		t.Fatal("unauthenticated projection accepted or returned partial output")
	}
}

func TestPublishIsCreateOnlyAndCompleteLast(t *testing.T) {
	r := filledResult()
	destination := filepath.Join(t.TempDir(), "bundle")
	if err := Publish(destination, r); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(destination, "COMPLETE.sha256"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"construction.g1", "project-v1.seme", "inventory-v2.seme", "package-v2.seme", "project-v3.seme", "dependency-v1.seme", "project-v4.seme", "package-v3.seme", "project-v5.seme"} {
		if _, err = os.Stat(filepath.Join(destination, name)); err != nil {
			t.Fatal(err)
		}
		if !contains(manifest, []byte(name+" ")) {
			t.Fatalf("manifest omitted %s", name)
		}
	}
	if err = Publish(destination, r); err == nil {
		t.Fatal("existing destination overwritten")
	}
	bad := r
	bad.ProjectV5 = nil
	badDestination := filepath.Join(filepath.Dir(destination), "bad")
	if err = Publish(badDestination, bad); err == nil {
		t.Fatal("empty artifact accepted")
	}
	if _, err = os.Stat(badDestination); !os.IsNotExist(err) {
		t.Fatal("partial destination left behind")
	}
}

func filledResult() Result {
	b := goprojectpipeline.Result{CanonicalG1: []byte("g1"), ProjectV1: []byte("p1"), InventoryV2: []byte("i2"), PackageV2: []byte("k2"), ProjectV3: []byte("p3")}
	return Result{Base: goupb03pipeline.Result{Base: b, DependencyV1: []byte("d1"), ProjectV4: []byte("p4")}, PackageV3: []byte("k3"), ProjectV5: []byte("p5")}
}

func contains(haystack, needle []byte) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
