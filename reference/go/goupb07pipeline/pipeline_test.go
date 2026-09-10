package goupb07pipeline

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"seme.local/reference/goresourceadapter"
	"testing"
)

func TestBuildRejectsUnauthenticatedWithoutPartial(t *testing.T) {
	r, err := Build(t.Context(), Input{})
	if err == nil || len(r.Durable) != 0 || len(r.ProjectV10) != 0 {
		t.Fatal("accepted unauthenticated")
	}
}
func TestPublishCreateOnlyPreservesDetachedResources(t *testing.T) {
	r := Result{Durable: []byte("durable"), Presentation: []byte("presentation"), ProjectV10: []byte("v10")}
	for _, f := range artifacts(r) {
		if f.name != "durable-state-v1.seme" && f.name != "source-presentation-v1.seme" && f.name != "project-v10.seme" {
			set(&r, f.name, []byte(f.name))
		}
	}
	data, second := []byte("resource"), []byte{0, 0xff, 0x53, 0x45}
	sum, secondSum := sha256.Sum256(data), sha256.Sum256(second)
	r.Base.Store.Blobs = []goresourceadapter.Blob{{SHA256: sum, Bytes: data}, {SHA256: secondSum, Bytes: second}}
	d := filepath.Join(t.TempDir(), "bundle")
	if err := Publish(d, r); err != nil {
		t.Fatal(err)
	}
	m, err := os.ReadFile(filepath.Join(d, "COMPLETE.sha256"))
	if err != nil || !bytes.Contains(m, []byte("durable-state-v1.seme ")) || !bytes.Contains(m, []byte("source-presentation-v1.seme ")) || !bytes.Contains(m, []byte("project-v10.seme ")) || !bytes.Contains(m, []byte("blobs/")) {
		t.Fatal("manifest", err)
	}
	if bytes.Count(m, []byte("\n")) != 17 { // header + 14 artifacts + two blobs
		t.Fatalf("completion entries=%q", m)
	}
	entries, err := os.ReadDir(d)
	if err != nil || len(entries) != 16 { // 14 artifacts, COMPLETE, blobs directory
		t.Fatalf("bundle shape entries=%d err=%v", len(entries), err)
	}
	blobs, err := os.ReadDir(filepath.Join(d, "blobs"))
	if err != nil || len(blobs) != 2 {
		t.Fatalf("blob count=%d err=%v", len(blobs), err)
	}
	repeat := filepath.Join(filepath.Dir(d), "repeat")
	if err = Publish(repeat, r); err != nil {
		t.Fatal(err)
	}
	m2, err := os.ReadFile(filepath.Join(repeat, "COMPLETE.sha256"))
	if err != nil || !bytes.Equal(m, m2) {
		t.Fatal("nondeterministic completion manifest", err)
	}
	if err = Publish(d, r); err == nil {
		t.Fatal("overwrote")
	}
	bad := r
	bad.Durable = nil
	p := filepath.Join(filepath.Dir(d), "partial")
	if err = Publish(p, bad); err == nil {
		t.Fatal("partial accepted")
	}
	if _, err = os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("partial remained")
	}
	missingPresentation := r
	missingPresentation.Presentation = nil
	missingPath := filepath.Join(filepath.Dir(d), "missing-presentation")
	if err = Publish(missingPath, missingPresentation); err == nil {
		t.Fatal("accepted missing presentation")
	}
	if _, err = os.Stat(missingPath); !os.IsNotExist(err) {
		t.Fatal("missing-presentation partial remained")
	}
	forged := r
	forged.Base.Store.Blobs = append([]goresourceadapter.Blob(nil), r.Base.Store.Blobs...)
	forged.Base.Store.Blobs[0].Bytes = []byte("forged")
	badBlob := filepath.Join(filepath.Dir(d), "bad-blob")
	if err = Publish(badBlob, forged); err == nil {
		t.Fatal("accepted forged blob")
	}
	if _, err = os.Stat(badBlob); !os.IsNotExist(err) {
		t.Fatal("forged partial remained")
	}
}
func set(r *Result, n string, b []byte) {
	switch n {
	case "construction-v36.g1":
		r.Base.Base.Construction = b
	case "execution-v36.seme":
		r.Base.Base.Execution = b
	case "project-base-v8.seme":
		r.Base.Base.ProjectBase = b
	case "inventory-v8.seme":
		r.Base.Base.Inventory = b
	case "package-detail-v4.seme":
		r.Base.Base.PackageDetail = b
	case "package-v4.seme":
		r.Base.Base.PackageV4 = b
	case "dependency-v1.seme":
		r.Base.Base.Dependency = b
	case "configuration-v3.seme":
		r.Base.Base.ConfigurationV3 = b
	case "project-v8.seme":
		r.Base.Base.ProjectV8 = b
	case "resource-v1.seme":
		r.Base.Resource = b
	case "project-v9.seme":
		r.Base.ProjectV9 = b
	}
}
