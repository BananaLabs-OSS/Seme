package resolutionfidelity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"seme.local/reference/projectsource"
)

func snapshot(t *testing.T) projectsource.Snapshot {
	t.Helper()
	root := t.TempDir()
	write := func(path, body string) {
		full := filepath.Join(root, filepath.FromSlash(path))
		if e := os.MkdirAll(filepath.Dir(full), 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(full, []byte(body), 0600); e != nil {
			t.Fatal(e)
		}
	}
	write("main.go", "package main")
	write("ignored/cache", "cache")
	write("generated.go", "// generated\npackage main")
	write("vendor/x.go", "package x")
	write("asset.bin", "opaque")
	out, e := projectsource.Discover(root, "example.test/evidence", projectsource.Toolchain{Language: "go", Toolchain: "go1.26", Profile: "go-uab-v1", SemanticRevision: "provider-rev-1"}, projectsource.Policy{TrackedExtensions: []string{".go"}, IgnoredPrefixes: []string{"ignored/"}, VendoredPrefixes: []string{"vendor/"}, GeneratedHeader: []byte("// generated"), MaxFiles: 16, MaxFileBytes: 1024, MaxTotalBytes: 8192})
	if e != nil {
		t.Fatal(e)
	}
	return out
}
func TestEvidenceIsDeterministicAndExplicit(t *testing.T) {
	s := snapshot(t)
	a, e := Build(s, "UAB-v1")
	if e != nil {
		t.Fatal(e)
	}
	b, e := Build(s, "UAB-v1")
	if e != nil || string(a) != string(b) {
		t.Fatal("nondeterministic")
	}
	if e = Validate(s, "UAB-v1", a); e != nil {
		t.Fatal(e)
	}
	var decoded Evidence
	if e = json.Unmarshal(a, &decoded); e != nil {
		t.Fatal(e)
	}
	if len(decoded.Classifications) != 5 || decoded.Classifications[0].Fidelity != "exact" || decoded.Classifications[0].Scope != "declared-uab-provider-profile-only" || decoded.Classifications[3].Fidelity != "native-island-opaque" || decoded.Inventory.Wasm || !decoded.BehaviorProgram.SeparateTargetPlan {
		t.Fatalf("claims=%#v", decoded)
	}
}
func TestEvidenceRejectsClaimForgeryAndJSONAmbiguity(t *testing.T) {
	s := snapshot(t)
	source, e := Build(s, "UAB-v1")
	if e != nil {
		t.Fatal(e)
	}
	mutate := func(f func(*Evidence)) []byte {
		var x Evidence
		if e := json.Unmarshal(source, &x); e != nil {
			t.Fatal(e)
		}
		f(&x)
		b, _ := json.Marshal(x)
		return append(b, '\n')
	}
	tests := map[string][]byte{
		"tracked-broadened": mutate(func(x *Evidence) { x.Classifications[0].Scope = "all-go" }),
		"vendored-exact":    mutate(func(x *Evidence) { x.Classifications[3].Fidelity = "exact" }),
		"inventory-wasm":    mutate(func(x *Evidence) { x.Inventory.Wasm = true }),
		"program-conflated": mutate(func(x *Evidence) { x.BehaviorProgram.SeparateTargetPlan = false }),
		"wrong-profile":     mutate(func(x *Evidence) { x.UABProfile = "other" }),
		"unknown-field":     append(source[:len(source)-2], []byte(",\"extra\":true}\n")...),
		"noncanonical":      append([]byte(" \n"), source...),
		"trailing":          append(append([]byte{}, source...), []byte("{}\n")...),
	}
	for name, b := range tests {
		t.Run(name, func(t *testing.T) {
			if Validate(s, "UAB-v1", b) == nil {
				t.Fatal("accepted")
			}
		})
	}
}
