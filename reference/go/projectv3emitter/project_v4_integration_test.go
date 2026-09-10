package projectv3emitter

import (
	"bytes"
	"os"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/dependencyemitter"
	"seme.local/reference/dependencyresolution"
	"seme.local/reference/projectdependencyinstance"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/projectv4emitter"
	"seme.local/reference/wire"
)

func v4Fixture(t *testing.T) (projectv4emitter.Input, projectdependencyinstance.Inputs) {
	t.Helper()
	v3in := fixture(t)
	v3, err := Emit(v3in)
	if err != nil {
		t.Fatal(err)
	}
	read := func(path string) []byte {
		b, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	contracts, err := contractcatalog.ResolveProjectContractSetV4(
		read("../../../modules/execution/v35/module.seme"),
		read("../../../modules/package/v2/module.seme"),
		read("../../../modules/dependency/v1/module.seme"),
		read("../../../modules/project/v4/module.seme"),
	)
	if err != nil {
		t.Fatal(err)
	}
	closure := dependencyresolution.Closure{
		Requirements: []dependencyresolution.Requirement{{Identity: "example.test/library", Requirement: "v1.2.3", Kind: dependencyresolution.External}, {Identity: "example.test/v3", Requirement: "source:abc", Kind: dependencyresolution.Local, Metadata: []dependencyresolution.Metadata{{Key: "direct", Value: "true"}}}},
		Entries:      []dependencyresolution.Entry{{Identity: "example.test/library", Ecosystem: "go", Version: "v1.2.3", Integrity: "h1:remote", IntegrityAlgorithm: "go-h1-tree", Source: "cache", SourceKind: "module-tree", Digest: "def", Kind: dependencyresolution.External}, {Identity: "example.test/v3", Version: "source:abc", Integrity: "h1:local", IntegrityAlgorithm: "go-h1-tree", Source: "local", SourceKind: "tree", Digest: "abc", Kind: dependencyresolution.Local, Dependencies: []string{"example.test/library"}}},
	}
	dependency, err := dependencyemitter.Emit(contracts.Dependency(), closure)
	if err != nil {
		t.Fatal(err)
	}
	input := projectv4emitter.Input{Contracts: contracts, ProjectV3: projectgraphInputs(v3in, v3), Dependency: dependency}
	return input, projectdependencyinstance.Inputs{Contracts: contracts, ProjectV3: input.ProjectV3, Dependency: dependency}
}

func projectgraphInputs(in Input, composed []byte) projectgraphinstance.Inputs {
	return projectgraphinstance.Inputs{Contracts: in.Contracts, ProjectV2: in.ProjectV2, Project: in.Project, Inventory: in.Inventory, PackageGraph: in.PackageGraph, Composed: composed}
}

func TestProjectV4EmitDeterministicValidatedImmutable(t *testing.T) {
	in, validation := v4Fixture(t)
	projectBefore, dependencyBefore := append([]byte(nil), in.ProjectV3.Composed...), append([]byte(nil), in.Dependency...)
	a, err := projectv4emitter.Emit(in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := projectv4emitter.Emit(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("v4 emission is nondeterministic")
	}
	if !bytes.Equal(projectBefore, in.ProjectV3.Composed) || !bytes.Equal(dependencyBefore, in.Dependency) {
		t.Fatal("input mutated")
	}
	validation.Composed = a
	if err = projectdependencyinstance.Validate(validation); err != nil {
		t.Fatal(err)
	}
	e, err := wire.Decode(a)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, q := range e.Entities {
		counts[q.Schema.String()]++
	}
	if counts[id("e019").String()] != 1 || counts[id("e018").String()] != 1 || counts[id("f010").String()] != 1 {
		t.Fatalf("root counts %#v", counts)
	}
}

func TestProjectV4RejectsTamperOrphanPinAndUnvalidatedInput(t *testing.T) {
	in, validation := v4Fixture(t)
	out, err := projectv4emitter.Emit(in)
	if err != nil {
		t.Fatal(err)
	}
	base, err := wire.Decode(out)
	if err != nil {
		t.Fatal(err)
	}

	tests := []wire.Envelope{}
	tamper := cloneEnvelope(t, base)
	for x, q := range tamper.Entities {
		if q.Schema == id("e019") {
			q.Fields[id("e192")] = blob(bytes.Repeat([]byte{7}, 32))
			tamper.Entities[x] = q
		}
	}
	tests = append(tests, tamper)
	orphan := cloneEnvelope(t, base)
	oid := id("7711")
	orphan.Entities[oid] = wire.Entity{ID: oid, Schema: id("9010"), Version: 1, Fields: map[wire.ID]wire.Value{}}
	tests = append(tests, orphan)
	wrongPin := cloneEnvelope(t, base)
	badRevision := id("f002")
	for x, q := range wrongPin.Entities {
		if q.Schema == id("13") && q.Fields[id("130")].Reference == id("f000") {
			q.Fields[id("131")] = blob(badRevision[:])
			wrongPin.Entities[x] = q
		}
	}
	tests = append(tests, wrongPin)
	for i, e := range tests {
		raw, er := wire.Encode(e)
		if er != nil {
			t.Fatal(er)
		}
		v := validation
		v.Composed = raw
		if er = projectdependencyinstance.Validate(v); er == nil {
			t.Fatalf("adversary %d accepted", i)
		}
	}

	bad := in
	bad.Contracts = contractcatalog.ProjectContractSetV4{}
	if got, er := projectv4emitter.Emit(bad); er == nil || got != nil {
		t.Fatalf("unvalidated contracts output=%x err=%v", got, er)
	}
	bad = in
	bad.Dependency = append([]byte(nil), bad.Dependency...)
	bad.Dependency[len(bad.Dependency)-1] ^= 1
	if got, er := projectv4emitter.Emit(bad); er == nil || got != nil {
		t.Fatalf("dependency tamper output=%x err=%v", got, er)
	}
	bad = in
	bad.ProjectV3.Composed = append([]byte(nil), bad.ProjectV3.Composed...)
	bad.ProjectV3.Composed[len(bad.ProjectV3.Composed)-1] ^= 1
	if got, er := projectv4emitter.Emit(bad); er == nil || got != nil {
		t.Fatalf("project tamper output=%x err=%v", got, er)
	}
}

func cloneEnvelope(t *testing.T, e wire.Envelope) wire.Envelope {
	t.Helper()
	b, err := wire.Encode(e)
	if err != nil {
		t.Fatal(err)
	}
	out, err := wire.Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
