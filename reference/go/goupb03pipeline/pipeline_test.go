package goupb03pipeline

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/dependencyinstance"
	"seme.local/reference/goprojectpipeline"
	"seme.local/reference/goprovider"
	"seme.local/reference/projectdependencyinstance"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/projectsource"
)

func TestBuildDeterministicValidatedUPB03(t *testing.T) {
	in := fixture(t)
	a, err := Build(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Build(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.DependencyV1, b.DependencyV1) || !bytes.Equal(a.ProjectV4, b.ProjectV4) {
		t.Fatal("nondeterministic")
	}
	if _, err = dependencyinstance.Validate(in.Contracts.Dependency(), a.DependencyV1); err != nil {
		t.Fatal(err)
	}
	v3 := projectgraphinstance.Inputs{Contracts: in.Base.Contracts.V3, ProjectV2: in.Base.Contracts.V2.Project(), Project: a.Base.ProjectV1, Inventory: a.Base.InventoryV2, PackageGraph: a.Base.PackageV2, Composed: a.Base.ProjectV3}
	if err = projectdependencyinstance.Validate(projectdependencyinstance.Inputs{Contracts: in.Contracts, ProjectV3: v3, Dependency: a.DependencyV1, Composed: a.ProjectV4}); err != nil {
		t.Fatal(err)
	}
	a.ProjectV4[0] ^= 1
	a.Base.Resolution.Packages[0].Files[0] = "forged"
	again, err := Build(context.Background(), in)
	if err != nil || again.ProjectV4[0] != 'S' {
		t.Fatal("aliased result")
	}
	for _, p := range again.Base.Resolution.Packages {
		for _, f := range p.Files {
			if f == "forged" {
				t.Fatal("aliased resolution")
			}
		}
	}
}

func TestRejectsDependencyAndContractFailuresAtomically(t *testing.T) {
	in := fixture(t)
	in.Dependency.Version = "latest"
	if out, err := Build(context.Background(), in); err == nil || len(out.ProjectV4) != 0 {
		t.Fatal("floating version accepted")
	}
	in = fixture(t)
	in.Contracts = contractcatalog.ProjectContractSetV4{}
	if out, err := Build(context.Background(), in); err == nil || len(out.DependencyV1) != 0 {
		t.Fatal("untrusted contracts accepted")
	}
	in = fixture(t)
	in.Base.Contracts.V3 = in.Base.Contracts.V1
	if out, err := Build(context.Background(), in); err == nil || len(out.ProjectV4) != 0 {
		t.Fatal("mixed contract generations accepted")
	}
	in = fixture(t)
	in.Dependency.LocalFrom = in.Dependency.LocalTo
	if out, err := Build(context.Background(), in); err == nil || len(out.ProjectV4) != 0 {
		t.Fatal("invalid local edge accepted")
	}
}

func TestPublishCompleteCreateOnlyBundle(t *testing.T) {
	in := fixture(t)
	r, err := Build(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "bundle")
	if err = Publish(dest, r); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"construction.g1", "project-v1.seme", "inventory-v2.seme", "package-v2.seme", "project-v3.seme", "dependency-v1.seme", "project-v4.seme", "COMPLETE.sha256"} {
		b, e := os.ReadFile(filepath.Join(dest, name))
		if e != nil || len(b) == 0 {
			t.Fatalf("%s: %v", name, e)
		}
	}
	before, _ := os.ReadFile(filepath.Join(dest, "project-v4.seme"))
	if err = Publish(dest, r); err == nil {
		t.Fatal("replaced bundle")
	}
	after, _ := os.ReadFile(filepath.Join(dest, "project-v4.seme"))
	if !bytes.Equal(before, after) {
		t.Fatal("existing bundle changed")
	}
	bad := filepath.Join(filepath.Dir(dest), "bad")
	r.ProjectV4 = nil
	if err = Publish(bad, r); err == nil {
		t.Fatal("empty artifact accepted")
	}
	if _, err = os.Lstat(bad); !os.IsNotExist(err) {
		t.Fatal("partial bundle remains")
	}
}

func fixture(t *testing.T) Input {
	t.Helper()
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	repo, err := filepath.Abs("../../../")
	if err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(repo, "fixtures/go-upb03-dependency-v1")
	proxy := filepath.Join(repo, "fixtures/go-upb03-offline-proxy")
	ec := read(filepath.Join(repo, "modules/execution/v35/module.seme"))
	p1 := read(filepath.Join(repo, "modules/package/v1/module.seme"))
	v1, err := contractcatalog.ResolveProjectContractSet(ec, p1, read(filepath.Join(repo, "modules/project/v1/module.seme")))
	if err != nil {
		t.Fatal(err)
	}
	v2, err := contractcatalog.ResolveProjectContractSetV2(ec, p1, read(filepath.Join(repo, "modules/project/v2/module.seme")))
	if err != nil {
		t.Fatal(err)
	}
	p2 := read(filepath.Join(repo, "modules/package/v2/module.seme"))
	v3, err := contractcatalog.ResolveProjectContractSetV3(ec, p2, read(filepath.Join(repo, "modules/project/v3/module.seme")))
	if err != nil {
		t.Fatal(err)
	}
	v4, err := contractcatalog.ResolveProjectContractSetV4(ec, p2, read(filepath.Join(repo, "modules/dependency/v1/module.seme")), read(filepath.Join(repo, "modules/project/v4/module.seme")))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, name := range []string{"application/application.go", "model/value.go", "policy/score.go"} {
		b, e := os.ReadFile(filepath.Join(project, name))
		if e != nil {
			t.Fatal(e)
		}
		files[name] = string(b)
	}
	sources, err := projectsource.Discover(project, "example.test/go-upb03-dependency-v1", projectsource.Toolchain{Language: "go", Toolchain: "go1.26", Profile: "bounded-v1", SemanticRevision: "provider-v1"}, projectsource.Policy{TrackedExtensions: []string{".go"}, IgnoredSuffixes: []string{"_test.go"}, GeneratedHeader: []byte("// generated"), MaxFiles: 32, MaxFileBytes: 1 << 20, MaxTotalBytes: 4 << 20})
	if err != nil {
		t.Fatal(err)
	}
	compile := func(ctx context.Context, input []byte) ([]byte, error) {
		dir := t.TempDir()
		src, out := filepath.Join(dir, "in.g1"), filepath.Join(dir, "out.seme")
		if e := os.WriteFile(src, input, 0600); e != nil {
			return nil, e
		}
		cmd := exec.CommandContext(ctx, filepath.Join(repo, "bootstrap/seme-k0-linux-amd64"), filepath.Join(repo, "compiler/g1-compiler.k0"), src, out)
		if b, e := cmd.CombinedOutput(); e != nil {
			return nil, errors.New(string(b))
		}
		return os.ReadFile(out)
	}
	app := "example.test/go-upb03-dependency-v1/application"
	model := "example.test/go-upb03-dependency-v1/model"
	documents := goprovider.DocumentSnapshot{Revision: 1, ModulePath: "example.test/go-upb03-dependency-v1", PackagePath: app, Entry: "Apply", Files: files}
	executionG1 := read(filepath.Join(repo, "modules/execution/v35/module.g1"))
	session, e := goprovider.NewIncrementalSession(executionG1)
	if e != nil {
		t.Fatal(e)
	}
	lifted := session.Apply(documents)
	if !lifted.Valid {
		t.Fatalf("fixture provider diagnostics: %#v", lifted.Diagnostics)
	}
	return Input{Base: goprojectpipeline.Input{Documents: documents, Sources: sources, Contracts: goprojectpipeline.Contracts{V1: v1, V2: v2, V3: v3}, ExecutionG1: executionG1, Compile: compile}, Contracts: v4, Dependency: DependencyInput{ProjectRoot: project, ProxyRoot: proxy, Module: "example.test/seme/checksum", Version: "v1.2.3", LocalFrom: app, LocalTo: model}}
}
