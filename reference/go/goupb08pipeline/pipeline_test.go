package goupb08pipeline

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"seme.local/reference/goresourceadapter"
)

func TestBuildRejectsUnauthenticatedWithoutPartial(t *testing.T) {
	r, err := Build(t.Context(), Input{})
	if err == nil || len(r.Transport) != 0 || len(r.ProjectV11) != 0 {
		t.Fatal("accepted unauthenticated")
	}
}
func TestPublishClosedDeterministicBundle(t *testing.T) {
	r := Result{Transport: []byte("transport"), ProjectV11: []byte("v11")}
	for _, f := range Artifacts(r) {
		if f.name != "ordered-transport-v1.seme" && f.name != "project-v11.seme" {
			set(&r, f.name, []byte(f.name))
		}
	}
	b := []byte("detached")
	sum := sha256.Sum256(b)
	r.Base.Base.Store.Blobs = []goresourceadapter.Blob{{SHA256: sum, Bytes: b}}
	d := filepath.Join(t.TempDir(), "bundle")
	if err := Publish(d, r); err != nil {
		t.Fatal(err)
	}
	m, err := os.ReadFile(filepath.Join(d, "COMPLETE.sha256"))
	if err != nil || !bytes.HasPrefix(m, []byte("seme-go-upb08-bundle-v1\n")) || bytes.Count(m, []byte("\n")) != 18 {
		t.Fatal("manifest", err)
	}
	a, err := os.ReadDir(d)
	if err != nil || len(a) != 18 {
		t.Fatalf("shape %d %v", len(a), err)
	}
	if err = Publish(d, r); err == nil {
		t.Fatal("overwrote")
	}
	bad := r
	bad.Transport = nil
	p := filepath.Join(filepath.Dir(d), "partial")
	if Publish(p, bad) == nil {
		t.Fatal("accepted missing")
	}
	if _, err = os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("partial remained")
	}
}
func set(r *Result, n string, b []byte) {
	switch n {
	case "construction-v36.g1":
		r.Base.Base.Base.Construction = b
	case "execution-v36.seme":
		r.Base.Base.Base.Execution = b
	case "project-base-v8.seme":
		r.Base.Base.Base.ProjectBase = b
	case "inventory-v8.seme":
		r.Base.Base.Base.Inventory = b
	case "package-detail-v4.seme":
		r.Base.Base.Base.PackageDetail = b
	case "package-v4.seme":
		r.Base.Base.Base.PackageV4 = b
	case "dependency-v1.seme":
		r.Base.Base.Base.Dependency = b
	case "configuration-v3.seme":
		r.Base.Base.Base.ConfigurationV3 = b
	case "project-v8.seme":
		r.Base.Base.Base.ProjectV8 = b
	case "resource-v1.seme":
		r.Base.Base.Resource = b
	case "project-v9.seme":
		r.Base.Base.ProjectV9 = b
	case "durable-state-v1.seme":
		r.Base.Durable = b
	case "source-presentation-v1.seme":
		r.Base.Presentation = b
	case "project-v10.seme":
		r.Base.ProjectV10 = b
	}
}
