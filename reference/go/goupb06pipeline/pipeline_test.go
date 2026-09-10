package goupb06pipeline

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
	if err == nil || len(r.Resource) != 0 || len(r.ProjectV9) != 0 {
		t.Fatal("accepted unauthenticated input")
	}
}
func TestPublishCreateOnlyCompleteLast(t *testing.T) {
	r := Result{Resource: []byte("resource"), ProjectV9: []byte("v9")}
	for _, f := range artifacts(r) {
		switch f.name {
		case "resource-v1.seme", "project-v9.seme":
		default:
			set(&r, f.name, []byte(f.name))
		}
	}
	b := []byte("detached")
	sum := sha256.Sum256(b)
	r.Store.Blobs = []goresourceadapter.Blob{{SHA256: sum, Bytes: b}}
	d := filepath.Join(t.TempDir(), "bundle")
	if err := Publish(d, r); err != nil {
		t.Fatal(err)
	}
	m, err := os.ReadFile(filepath.Join(d, "COMPLETE.sha256"))
	if err != nil || !bytes.Contains(m, []byte("resource-v1.seme ")) || !bytes.Contains(m, []byte("blobs/")) {
		t.Fatal("incomplete manifest", err)
	}
	if got, err := os.ReadFile(filepath.Join(d, "blobs", fmtHex(sum))); err != nil || !bytes.Equal(got, b) {
		t.Fatal("blob", err)
	}
	if err = Publish(d, r); err == nil {
		t.Fatal("overwrote bundle")
	}
	bad := r
	bad.Resource = nil
	p := filepath.Join(filepath.Dir(d), "partial")
	if err = Publish(p, bad); err == nil {
		t.Fatal("accepted partial")
	}
	if _, err = os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("partial remained")
	}
	mut := r
	mut.Store.Blobs[0].Bytes = []byte("wrong")
	if err = Publish(filepath.Join(filepath.Dir(d), "bad"), mut); err == nil {
		t.Fatal("accepted forged blob")
	}
}
func fmtHex(s [32]byte) string {
	const h = "0123456789abcdef"
	b := make([]byte, 64)
	for i, x := range s {
		b[i*2] = h[x>>4]
		b[i*2+1] = h[x&15]
	}
	return string(b)
}
func set(r *Result, n string, b []byte) {
	switch n {
	case "construction-v36.g1":
		r.Base.Construction = b
	case "execution-v36.seme":
		r.Base.Execution = b
	case "project-base-v8.seme":
		r.Base.ProjectBase = b
	case "inventory-v8.seme":
		r.Base.Inventory = b
	case "package-detail-v4.seme":
		r.Base.PackageDetail = b
	case "package-v4.seme":
		r.Base.PackageV4 = b
	case "dependency-v1.seme":
		r.Base.Dependency = b
	case "configuration-v3.seme":
		r.Base.ConfigurationV3 = b
	case "project-v8.seme":
		r.Base.ProjectV8 = b
	}
}
