package projectsource

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

func policy() Policy {
	return Policy{TrackedExtensions: []string{".go"}, IgnoredPrefixes: []string{"ignored/"}, IgnoredSuffixes: []string{"_test.go"}, VendoredPrefixes: []string{"vendor/"}, GeneratedHeader: []byte("// Code generated "), MaxFiles: 32, MaxFileBytes: 1024, MaxTotalBytes: 4096}
}
func toolchain() Toolchain {
	return Toolchain{Language: "go", Toolchain: "go1.26", Profile: "linux-amd64/bounded-v1", SemanticRevision: "go-bounded-v1"}
}
func write(t *testing.T, root, path, body string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverAllFiveClassesDeterministically(t *testing.T) {
	root := t.TempDir()
	write(t, root, "z.bin", "opaque")
	write(t, root, "vendor/lib.go", "package lib")
	write(t, root, "ignored/cache", "ignored")
	write(t, root, "generated.go", "// Code generated tool; DO NOT EDIT.\npackage p")
	write(t, root, "a.go", "package p")
	first, err := Discover(root, "example/project", toolchain(), policy())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Discover(root, "example/project", toolchain(), policy())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("discovery differs")
	}
	want := []Class{Tracked, Generated, Ignored, Vendored, Opaque}
	if len(first.Units) != len(want) {
		t.Fatal(first.Units)
	}
	for i, x := range want {
		if first.Units[i].Class != x {
			t.Fatalf("unit %d = %s", i, first.Units[i].Class)
		}
		wantPreservation := ByteExact
		if x == Tracked {
			wantPreservation = SemanticProjection
		}
		if first.Units[i].Preservation != wantPreservation {
			t.Fatalf("unit %d preservation = %s", i, first.Units[i].Preservation)
		}
	}
	for i := 1; i < len(first.Units); i++ {
		if first.Units[i-1].Path >= first.Units[i].Path {
			t.Fatal("not ordered")
		}
	}
	if len(first.ContentRevision) != 64 {
		t.Fatal(first.ContentRevision)
	}
	if err := Verify(root, first, policy()); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyRejectsDigestDrift(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.go", "package p")
	snapshot, err := Discover(root, "p", toolchain(), policy())
	if err != nil {
		t.Fatal(err)
	}
	write(t, root, "a.go", "package changed")
	if err := Verify(root, snapshot, policy()); err == nil || !strings.Contains(err.Error(), "digest_drift") {
		t.Fatalf("err=%v", err)
	}
}

func TestRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.go", "package p")
	if err := os.Symlink("a.go", filepath.Join(root, "alias.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := Discover(root, "p", toolchain(), policy()); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("err=%v", err)
	}
}

func TestRejectsSymlinkInRootPath(t *testing.T) {
	parent := t.TempDir()
	realRoot := filepath.Join(parent, "real")
	if err := os.Mkdir(realRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, realRoot, "a.go", "package p")
	linkedRoot := filepath.Join(parent, "linked")
	if err := os.Symlink(realRoot, linkedRoot); err != nil {
		t.Fatal(err)
	}
	if _, err := Discover(linkedRoot, "p", toolchain(), policy()); err == nil || !strings.Contains(err.Error(), "root_symlink_component") {
		t.Fatalf("err=%v", err)
	}
}

func TestRejectsNonregularFile(t *testing.T) {
	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, "pipe"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Discover(root, "p", toolchain(), policy()); err == nil || !strings.Contains(err.Error(), "nonregular") {
		t.Fatalf("err=%v", err)
	}
}

func TestRejectsPolicyTraversalAndAmbiguity(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.go", "package p")
	tests := map[string]Policy{"traversal": func() Policy { p := policy(); p.IgnoredPrefixes = []string{"../escape/"}; return p }(), "same-prefix": func() Policy { p := policy(); p.VendoredPrefixes = []string{"ignored/"}; return p }(), "overlap": func() Policy { p := policy(); p.VendoredPrefixes = []string{"ignored/nested/"}; return p }(), "extension": func() Policy { p := policy(); p.TrackedExtensions = []string{".go", ".go"}; return p }()}
	tests["suffix"] = func() Policy { p := policy(); p.IgnoredSuffixes = []string{"_test.go", "_test.go"}; return p }()
	for name, p := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Discover(root, "p", toolchain(), p); err == nil {
				t.Fatal("accepted invalid policy")
			}
		})
	}
}

func TestRejectsBoundsAndIdentity(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.go", "package p")
	p := policy()
	p.MaxFiles = 0
	if _, err := Discover(root, "p", toolchain(), p); err == nil {
		t.Fatal("accepted bounds")
	}
	if _, err := Discover(root, "", toolchain(), policy()); err == nil {
		t.Fatal("accepted empty identity")
	}
	badToolchain := toolchain()
	badToolchain.Profile = ""
	if _, err := Discover(root, "p", badToolchain, policy()); err == nil {
		t.Fatal("accepted incomplete toolchain")
	}
}

func TestRejectsEmptyProject(t *testing.T) {
	if _, err := Discover(t.TempDir(), "p", toolchain(), policy()); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("err=%v", err)
	}
}
