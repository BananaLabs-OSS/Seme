package contractcatalog

import (
	"os"
	"strings"
	"testing"

	"seme.local/reference/wire"
)

func artifacts(t *testing.T) (execution, packages, project []byte) {
	t.Helper()
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	return read("../../../modules/execution/v35/module.seme"), read("../../../modules/package/v1/module.seme"), read("../../../modules/project/v1/module.seme")
}

func TestResolveProjectContractSet(t *testing.T) {
	e, p, r := artifacts(t)
	set, err := ResolveProjectContractSet(e, p, r)
	if err != nil {
		t.Fatal(err)
	}
	if !set.Validated() || len(set.Execution().Exports()) != 292 || len(set.Package().Exports()) != 25 || len(set.Project().Imports()) != 2 {
		t.Fatalf("unexpected resolved contracts")
	}
	copy := set.Project().Envelope()
	delete(copy.Entities, copy.Module)
	if _, exists := set.Project().Envelope().Entities[set.Project().Pin().Module]; !exists {
		t.Fatal("contract envelope accessor exposed mutable catalog state")
	}
}

func TestResolveRejectsUntrustedContractInputs(t *testing.T) {
	e, p, r := artifacts(t)
	tests := map[string]func() error{
		"raw-concatenation": func() error {
			_, err := ResolveProjectContractSet(e, p, append(append([]byte{}, r...), p...))
			return err
		},
		"malformed":          func() error { _, err := ResolveProjectContractSet(e, p, r[:len(r)-1]); return err },
		"wrong-substitution": func() error { _, err := ResolveProjectContractSet(e, p, p); return err },
	}
	for name, run := range tests {
		t.Run(name, func(t *testing.T) {
			if err := run(); err == nil {
				t.Fatal("accepted")
			}
		})
	}
}

func TestResolveRejectsMissingExportWrongPinAndOrphanImport(t *testing.T) {
	_, _, source := artifacts(t)
	base, err := wire.Decode(source)
	if err != nil {
		t.Fatal(err)
	}
	expect := Expectation{
		Pin: Pin{projectModule, projectRev}, ModuleVersion: 1,
		RequiredExports: []wire.ID{mustID("0000000000000000000000000000e010"), mustID("0000000000000000000000000000e011")},
		Imports:         []Pin{{packageModule, packageRev}, {executionModule, executionRev}},
	}
	clone := func() wire.Envelope { b, _ := wire.Encode(base); e, _ := wire.Decode(b); return e }
	t.Run("missing-export", func(t *testing.T) {
		e := clone()
		m := e.Entities[e.Module]
		list := m.Fields[fExports]
		list.List = list.List[1:]
		m.Fields[fExports] = list
		e.Entities[e.Module] = m
		b, _ := wire.Encode(e)
		if _, err := Resolve(b, expect); err == nil || !strings.Contains(err.Error(), "required_export") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("wrong-pin", func(t *testing.T) {
		e := clone()
		m := e.Entities[e.Module]
		iid := m.Fields[fImports].List[0].Reference
		imp := e.Entities[iid]
		value := imp.Fields[fImportRev]
		value.Bytes[15] ^= 1
		imp.Fields[fImportRev] = value
		e.Entities[iid] = imp
		b, _ := wire.Encode(e)
		if _, err := Resolve(b, expect); err == nil || !strings.Contains(err.Error(), "import_pins") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("orphan-import", func(t *testing.T) {
		e := clone()
		id := mustID("8000000000000000000000000000eeee")
		e.Entities[id] = wire.Entity{ID: id, Schema: importSchema, Version: 1, Fields: map[wire.ID]wire.Value{fImportMod: {Tag: 6, Reference: packageModule}, fImportRev: {Tag: 5, Bytes: packageRev[:]}}}
		b, _ := wire.Encode(e)
		if _, err := Resolve(b, expect); err == nil || !strings.Contains(err.Error(), "orphan_import") {
			t.Fatalf("got %v", err)
		}
	})
}

func TestResolveRejectsDigestMutationAndDuplicateTypedReference(t *testing.T) {
	_, _, source := artifacts(t)
	if _, err := Resolve(source, Expectation{Pin: Pin{projectModule, projectRev}, ModuleVersion: 1, Digest: mustDigest("0100000000000000000000000000000000000000000000000000000000000000")}); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("got %v", err)
	}
	e, _ := wire.Decode(source)
	m := e.Entities[e.Module]
	imports := m.Fields[fImports]
	imports.List = append(imports.List, imports.List[0])
	m.Fields[fImports] = imports
	e.Entities[e.Module] = m
	b, _ := wire.Encode(e)
	if _, err := Resolve(b, Expectation{Pin: Pin{projectModule, projectRev}, ModuleVersion: 1}); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("got %v", err)
	}
}
