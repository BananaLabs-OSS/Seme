package projectbuild

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprovider"
	"seme.local/reference/wire"
)

func testContracts(t *testing.T) contractcatalog.ProjectContractSet {
	t.Helper()
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	x, e := contractcatalog.ResolveProjectContractSet(read("../../../modules/execution/v35/module.seme"), read("../../../modules/package/v1/module.seme"), read("../../../modules/project/v1/module.seme"))
	if e != nil {
		t.Fatal(e)
	}
	return x
}

func frozenCompiler(t *testing.T) Compile {
	t.Helper()
	return func(ctx context.Context, input []byte) ([]byte, error) {
		dir := t.TempDir()
		in, out := filepath.Join(dir, "in.g1"), filepath.Join(dir, "out.seme")
		if err := os.WriteFile(in, input, 0600); err != nil {
			return nil, err
		}
		command := exec.CommandContext(ctx, "../../../bootstrap/seme-k0-linux-amd64", "../../../compiler/g1-compiler.k0", in, out)
		if b, err := command.CombinedOutput(); err != nil {
			return nil, errors.New(string(b))
		}
		return os.ReadFile(out)
	}
}

func TestBuildWithInjectedFrozenCompiler(t *testing.T) {
	g1, e := os.ReadFile("../../../modules/execution/v35/module.g1")
	if e != nil {
		t.Fatal(e)
	}
	compile := frozenCompiler(t)
	snapshot := goprovider.DocumentSnapshot{Revision: 7, ModulePath: "example.test/build", PackagePath: "example.test/build/app", Entry: "Apply", Files: map[string]string{
		"lib/value.go": `package lib
func addOne(v int64) int64 { return v + 1 }
func AddOne(v int64) int64 { return addOne(v) }`,
		"app/main.go": `package app
import "example.test/build/lib"
func Apply(v int64) int64 { return lib.AddOne(v) }`,
	}}
	result, e := Build(context.Background(), snapshot, testContracts(t), g1, compile)
	if e != nil {
		t.Fatal(e)
	}
	if len(result.Artifact) == 0 || len(result.CanonicalG1) == 0 || result.SourceDigest == "" || len(result.Packages) != 2 || len(result.Resolution.Packages) != 2 {
		t.Fatalf("result=%#v", result)
	}
	privateFound := false
	for _, p := range result.Packages {
		for _, member := range p.Members {
			if member.Name == "addOne" {
				privateFound = !member.Exported && member.Document == "lib/value.go" && member.Line > 0 && member.Column > 0
			}
		}
	}
	if !privateFound {
		t.Fatalf("private package member missing: %#v", result.Packages)
	}
	wantG1First := result.CanonicalG1[0]
	result.Artifact[0] ^= 1
	result.CanonicalG1[0] ^= 1
	result.Packages[0].Dependencies = append(result.Packages[0].Dependencies, "mutation")
	result.Packages[0].Members[0].Parameters[0] = "mutation"
	result.Resolution.Packages[0].Files[0] = "mutation"
	again, e := Build(context.Background(), snapshot, testContracts(t), g1, compile)
	if e != nil {
		t.Fatal(e)
	}
	if len(again.Artifact) == 0 || again.Artifact[0] != 'S' || len(again.CanonicalG1) == 0 || again.CanonicalG1[0] != wantG1First {
		t.Fatal("result was not copy safe")
	}
	for _, p := range again.Packages {
		for _, member := range p.Members {
			for _, parameter := range member.Parameters {
				if parameter == "mutation" {
					t.Fatal("member metadata was not copy safe")
				}
			}
		}
	}
	for _, p := range again.Resolution.Packages {
		for _, file := range p.Files {
			if file == "mutation" {
				t.Fatal("resolution metadata was not copy safe")
			}
		}
	}
}

func TestBuildOwnsCanonicalLogEffectWithoutSourceDependency(t *testing.T) {
	g1, err := os.ReadFile("../../../modules/execution/v35/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	snapshot := goprovider.DocumentSnapshot{Revision: 1, ModulePath: "example.test/effects", PackagePath: "example.test/effects", Entry: "Apply", Files: map[string]string{
		"main.go": "package effects\nimport \"log\"\nfunc Apply(v bool) bool { log.Print(v); return v }\n",
	}}
	result, err := Build(context.Background(), snapshot, testContracts(t), g1, frozenCompiler(t))
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := wire.Decode(result.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entity := range envelope.Entities {
		if entity.Schema != mustID("0000000000000000000000000000b010") {
			continue
		}
		effects := entity.Fields[mustID("0000000000000000000000000000b104")]
		found = effects.Tag == 7 && len(effects.List) == 1
	}
	if !found {
		t.Fatal("canonical log effect was not owned by its package")
	}
	for _, p := range result.Packages {
		if len(p.Dependencies) != 0 {
			t.Fatalf("standard-library effect became a source dependency: %#v", p.Dependencies)
		}
	}
}
func TestBuildNeverCompilesInvalidCurrentSnapshot(t *testing.T) {
	g1, e := os.ReadFile("../../../modules/execution/v35/module.g1")
	if e != nil {
		t.Fatal(e)
	}
	called := false
	_, err := Build(context.Background(), goprovider.DocumentSnapshot{Revision: 1, ModulePath: "example.test/x", PackagePath: "example.test/x", Entry: "Apply", Files: map[string]string{"main.go": "package x\nfunc Apply("}}, testContracts(t), g1, func(context.Context, []byte) ([]byte, error) { called = true; return nil, nil })
	if err == nil || called {
		t.Fatalf("err=%v called=%v", err, called)
	}
}
func TestBuildPropagatesCompilerFailureAndRejectsNonWire(t *testing.T) {
	g1, e := os.ReadFile("../../../modules/execution/v35/module.g1")
	if e != nil {
		t.Fatal(e)
	}
	snapshot := goprovider.DocumentSnapshot{Revision: 1, ModulePath: "example.test/x", PackagePath: "example.test/x", Entry: "Apply", Files: map[string]string{"main.go": "package x\nfunc Apply(v int64) int64 { return v + 1 }"}}
	sentinel := errors.New("compiler stopped")
	if _, err := Build(context.Background(), snapshot, testContracts(t), g1, func(context.Context, []byte) ([]byte, error) { return nil, sentinel }); !errors.Is(err, sentinel) {
		t.Fatalf("got %v", err)
	}
	if _, err := Build(context.Background(), snapshot, testContracts(t), g1, func(context.Context, []byte) ([]byte, error) { return []byte("not wire"), nil }); err == nil {
		t.Fatal("accepted non-wire compiler output")
	}
}
