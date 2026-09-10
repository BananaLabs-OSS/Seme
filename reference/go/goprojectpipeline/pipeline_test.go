package goprojectpipeline

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprovider"
	"seme.local/reference/projectsource"
)

func TestBuildDeterministicMultiPackageV3Chain(t *testing.T) {
	in := fixture(t)
	a, err := Build(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Build(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.ProjectV1, b.ProjectV1) || !bytes.Equal(a.InventoryV2, b.InventoryV2) || !bytes.Equal(a.PackageV2, b.PackageV2) || !bytes.Equal(a.ProjectV3, b.ProjectV3) {
		t.Fatal("pipeline is nondeterministic")
	}
	for _, raw := range [][]byte{a.InventoryV2, a.PackageV2, a.ProjectV3} {
		if bytes.Contains(raw, []byte("private implementation stays source-only")) {
			t.Fatal("raw source leaked into artifact")
		}
	}
	wantG1First := a.CanonicalG1[0]
	a.ProjectV3[0] ^= 1
	a.CanonicalG1[0] ^= 1
	a.Resolution.Packages[0].Files[0] = "forged.go"
	again, err := Build(context.Background(), in)
	if err != nil || again.ProjectV3[0] != 'S' || again.CanonicalG1[0] != wantG1First {
		t.Fatal("result aliases pipeline state")
	}
	for _, p := range again.Resolution.Packages {
		for _, name := range p.Files {
			if name == "forged.go" {
				t.Fatal("resolution result aliases pipeline state")
			}
		}
	}
}

func TestBuildRejectsSourceMismatchInvalidAndCompileFailure(t *testing.T) {
	in := fixture(t)
	in.Documents.Files["lib/value.go"] += "\n// drift"
	if _, err := Build(context.Background(), in); err == nil {
		t.Fatal("source/inventory drift accepted")
	}
	in = fixture(t)
	in.Documents.Files["app/main.go"] = "package app\nfunc Apply("
	called := false
	in.Compile = func(context.Context, []byte) ([]byte, error) { called = true; return nil, nil }
	if _, err := Build(context.Background(), in); err == nil || called {
		t.Fatal("invalid current source reached compiler")
	}
	in = fixture(t)
	sentinel := errors.New("stop")
	in.Compile = func(context.Context, []byte) ([]byte, error) { return nil, sentinel }
	if _, err := Build(context.Background(), in); !errors.Is(err, sentinel) {
		t.Fatalf("compile failure lost: %v", err)
	}
	in = fixture(t)
	in.Contracts.V3 = contractcatalog.ProjectContractSet{}
	if _, err := Build(context.Background(), in); err == nil {
		t.Fatal("untrusted v3 contracts accepted")
	}
}

func TestPublishCreateOnlyAndRollback(t *testing.T) {
	r := Result{CanonicalG1: []byte("g1"), ProjectV1: []byte("p1"), InventoryV2: []byte("i2"), PackageV2: []byte("p2"), ProjectV3: []byte("p3")}
	parent := t.TempDir()
	dest := filepath.Join(parent, "bundle")
	if err := Publish(dest, r); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(dest, "project-v3.seme"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(dest, "COMPLETE.sha256"))
	if err != nil || !bytes.Contains(manifest, []byte("project-v3.seme ")) {
		t.Fatalf("completion marker: %q %v", manifest, err)
	}
	if err = Publish(dest, Result{CanonicalG1: []byte("x"), ProjectV1: []byte("x"), InventoryV2: []byte("x"), PackageV2: []byte("x"), ProjectV3: []byte("x")}); err == nil {
		t.Fatal("existing nonempty destination replaced")
	}
	after, _ := os.ReadFile(filepath.Join(dest, "project-v3.seme"))
	if !bytes.Equal(before, after) {
		t.Fatal("existing destination changed")
	}
	empty := filepath.Join(parent, "empty")
	if err = os.Mkdir(empty, 0700); err != nil {
		t.Fatal(err)
	}
	if err = Publish(empty, r); err == nil {
		t.Fatal("existing empty destination replaced")
	}
	entries, _ := os.ReadDir(empty)
	if len(entries) != 0 {
		t.Fatal("existing empty destination changed")
	}
	bad := filepath.Join(parent, "bad")
	if err = Publish(bad, Result{ProjectV1: []byte("only")}); err == nil {
		t.Fatal("empty artifact accepted")
	}
	if _, err = os.Lstat(bad); !os.IsNotExist(err) {
		t.Fatal("failed publication left destination")
	}
}

func fixture(t *testing.T) Input {
	t.Helper()
	read := func(p string) []byte {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	execution := read("../../../modules/execution/v35/module.seme")
	p1 := read("../../../modules/package/v1/module.seme")
	v1, err := contractcatalog.ResolveProjectContractSet(execution, p1, read("../../../modules/project/v1/module.seme"))
	if err != nil {
		t.Fatal(err)
	}
	v2, err := contractcatalog.ResolveProjectContractSetV2(execution, p1, read("../../../modules/project/v2/module.seme"))
	if err != nil {
		t.Fatal(err)
	}
	v3, err := contractcatalog.ResolveProjectContractSetV3(execution, read("../../../modules/package/v2/module.seme"), read("../../../modules/project/v3/module.seme"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"model/offset.go": "package model\n// private implementation stays source-only\nfunc normalize(v int64) int64 { return v }\nfunc Offset(v int64) int64 { return normalize(v) + 1 }\n", "lib/value.go": "package lib\nimport \"example.test/pipeline/model\"\nfunc AddOne(v int64) int64 { return model.Offset(v) }\n", "app/main.go": "package app\nimport \"example.test/pipeline/lib\"\nfunc Apply(v int64) int64 { return lib.AddOne(v) }\n"}
	root := t.TempDir()
	for name, data := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	sources, err := projectsource.Discover(root, "example.test/pipeline", projectsource.Toolchain{Language: "go", Toolchain: "go1.26", Profile: "bounded-v1", SemanticRevision: "provider-v1"}, projectsource.Policy{TrackedExtensions: []string{".go"}, GeneratedHeader: []byte("// generated"), MaxFiles: 8, MaxFileBytes: 4096, MaxTotalBytes: 8192})
	if err != nil {
		t.Fatal(err)
	}
	compile := func(ctx context.Context, input []byte) ([]byte, error) {
		dir := t.TempDir()
		src, out := filepath.Join(dir, "in.g1"), filepath.Join(dir, "out.seme")
		if err := os.WriteFile(src, input, 0600); err != nil {
			return nil, err
		}
		cmd := exec.CommandContext(ctx, "../../../bootstrap/seme-k0-linux-amd64", "../../../compiler/g1-compiler.k0", src, out)
		if raw, err := cmd.CombinedOutput(); err != nil {
			return nil, errors.New(string(raw))
		}
		return os.ReadFile(out)
	}
	return Input{Documents: goprovider.DocumentSnapshot{Revision: 1, ModulePath: "example.test/pipeline", PackagePath: "example.test/pipeline/app", Entry: "Apply", Files: files}, Sources: sources, Contracts: Contracts{v1, v2, v3}, ExecutionG1: read("../../../modules/execution/v35/module.g1"), Compile: compile}
}
