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

func TestBuildWithInjectedFrozenCompiler(t *testing.T) {
	g1, e := os.ReadFile("../../../modules/execution/v35/module.g1")
	if e != nil {
		t.Fatal(e)
	}
	compile := func(ctx context.Context, input []byte) ([]byte, error) {
		dir := t.TempDir()
		in, out := filepath.Join(dir, "in.g1"), filepath.Join(dir, "out.seme")
		if e := os.WriteFile(in, input, 0600); e != nil {
			return nil, e
		}
		command := exec.CommandContext(ctx, "../../../bootstrap/seme-k0-linux-amd64", "../../../compiler/g1-compiler.k0", in, out)
		if b, e := command.CombinedOutput(); e != nil {
			return nil, errors.New(string(b))
		}
		return os.ReadFile(out)
	}
	snapshot := goprovider.DocumentSnapshot{Revision: 7, ModulePath: "example.test/build", PackagePath: "example.test/build/app", Entry: "Apply", Files: map[string]string{
		"lib/value.go": `package lib
func AddOne(v int64) int64 { return v + 1 }`,
		"app/main.go": `package app
import "example.test/build/lib"
func Apply(v int64) int64 { return lib.AddOne(v) }`,
	}}
	result, e := Build(context.Background(), snapshot, testContracts(t), g1, compile)
	if e != nil {
		t.Fatal(e)
	}
	if len(result.Artifact) == 0 || result.SourceDigest == "" || len(result.Packages) != 2 {
		t.Fatalf("result=%#v", result)
	}
	result.Artifact[0] ^= 1
	result.Packages[0].Dependencies = append(result.Packages[0].Dependencies, "mutation")
	again, e := Build(context.Background(), snapshot, testContracts(t), g1, compile)
	if e != nil {
		t.Fatal(e)
	}
	if len(again.Artifact) == 0 || again.Artifact[0] != 'S' {
		t.Fatal("result was not copy safe")
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
