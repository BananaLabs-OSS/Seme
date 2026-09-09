package projectbundle

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"seme.local/reference/projectsource"
)

func TestCaptureDetachedBundleAndRejectTampering(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{"a.go": "package a\n", "NOTICE": "opaque\n", "ignored/cache": "held\n"} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	policy := projectsource.Policy{TrackedExtensions: []string{".go"}, IgnoredPrefixes: []string{"ignored/"}, VendoredPrefixes: []string{"vendor/"}, GeneratedHeader: []byte("// Code generated "), MaxFiles: 8, MaxFileBytes: 1024, MaxTotalBytes: 4096}
	toolchain := projectsource.Toolchain{Language: "go", Toolchain: "go1.25", Profile: "bounded-v1", SemanticRevision: "policy-v1"}
	snapshot, err := projectsource.Discover(root, "example/project", toolchain, policy)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := Capture(root, snapshot, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root, root+"-moved"); err != nil {
		t.Fatal(err)
	}
	if err := Validate(snapshot, bundle); err != nil {
		t.Fatal(err)
	}
	for _, unit := range snapshot.Units {
		data, ok := bundle.Lookup(unit.SHA256)
		if !ok || int64(len(data)) != unit.Size {
			t.Fatalf("missing %s", unit.Path)
		}
	}

	bad := Bundle{Blobs: append([]Blob(nil), bundle.Blobs...)}
	bad.Blobs[0].Bytes = append([]byte(nil), bad.Blobs[0].Bytes...)
	bad.Blobs[0].Bytes[0] ^= 1
	if err := Validate(snapshot, bad); err == nil || !strings.Contains(err.Error(), "digest_mismatch") {
		t.Fatalf("tamper err=%v", err)
	}
	missing := Bundle{Blobs: append([]Blob(nil), bundle.Blobs[1:]...)}
	if err := Validate(snapshot, missing); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing err=%v", err)
	}
}

func TestValidateRejectsUnreferencedAndUnsortedBlobs(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	policy := projectsource.Policy{TrackedExtensions: []string{".go"}, GeneratedHeader: []byte("// generated"), MaxFiles: 2, MaxFileBytes: 1024, MaxTotalBytes: 1024}
	snapshot, err := projectsource.Discover(root, "p", projectsource.Toolchain{Language: "go", Toolchain: "go1.25", Profile: "v1", SemanticRevision: "v1"}, policy)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := Capture(root, snapshot, policy)
	if err != nil {
		t.Fatal(err)
	}
	extraBytes := []byte("extra")
	extraDigest := sha256.Sum256(extraBytes)
	extra := Blob{SHA256: hex.EncodeToString(extraDigest[:]), Bytes: extraBytes}
	bad := Bundle{Blobs: append(append([]Blob(nil), bundle.Blobs...), extra)}
	sort.Slice(bad.Blobs, func(i, j int) bool { return bad.Blobs[i].SHA256 < bad.Blobs[j].SHA256 })
	if err := Validate(snapshot, bad); err == nil || !strings.Contains(err.Error(), "unreferenced") {
		t.Fatalf("extra err=%v", err)
	}
	bad = Bundle{Blobs: []Blob{bundle.Blobs[0], bundle.Blobs[0]}}
	if err := Validate(snapshot, bad); err == nil || !strings.Contains(err.Error(), "order") {
		t.Fatalf("order err=%v", err)
	}
}
