package projectroundtrip

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"seme.local/reference/projectbundle"
	"seme.local/reference/projectsource"
)

func TestPublishProjectedUsesDetachedBundleAndPreservesRegions(t *testing.T) {
	source, snapshot := fixture(t)
	bundle, err := projectbundle.Capture(source, snapshot, testPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(source, source+"-unavailable"); err != nil {
		t.Fatal(err)
	}
	projected := map[string][]byte{"main.go": []byte("package main\n\nfunc projected() {}\n")}
	dest := filepath.Join(t.TempDir(), "projected")
	after, err := PublishProjected(dest, snapshot, bundle, projected, testPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if after.ContentRevision == snapshot.ContentRevision {
		t.Fatal("tracked source revision did not change")
	}
	for i, unit := range snapshot.Units {
		if unit.Class != projectsource.Tracked && unit != after.Units[i] {
			t.Fatalf("preserved unit changed: %s", unit.Path)
		}
	}
	got, err := os.ReadFile(filepath.Join(dest, "main.go"))
	if err != nil || string(got) != string(projected["main.go"]) {
		t.Fatalf("tracked projection: %v %q", err, got)
	}
}

func TestPublishProjectedRejectsSetsAndReclassificationWithoutPublication(t *testing.T) {
	source, snapshot := fixture(t)
	bundle, err := projectbundle.Capture(source, snapshot, testPolicy())
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]map[string][]byte{
		"missing": {},
		"extra":   {"main.go": []byte("package main\n"), "extra.go": []byte("package main\n")},
		"class":   {"main.go": []byte("// Code generated tool\npackage main\n")},
	}
	for name, projected := range tests {
		t.Run(name, func(t *testing.T) {
			dest := filepath.Join(t.TempDir(), "projected")
			_, err := PublishProjected(dest, snapshot, bundle, projected, testPolicy())
			if err == nil {
				t.Fatal("accepted")
			}
			if _, statErr := os.Lstat(dest); !os.IsNotExist(statErr) {
				t.Fatalf("partial destination: %v", statErr)
			}
		})
	}
	tampered := bundle
	tampered.Blobs = append([]projectbundle.Blob(nil), bundle.Blobs...)
	tampered.Blobs[0].Bytes = append([]byte(nil), tampered.Blobs[0].Bytes...)
	tampered.Blobs[0].Bytes[0] ^= 1
	dest := filepath.Join(t.TempDir(), "projected")
	_, err = PublishProjected(dest, snapshot, tampered, map[string][]byte{"main.go": []byte("package main\n")}, testPolicy())
	if err == nil || !strings.Contains(err.Error(), "digest_mismatch") {
		t.Fatalf("tamper err=%v", err)
	}
}
