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

func TestResolveProjectContractSetV2(t *testing.T) {
	e, p, _ := artifacts(t)
	v2, err := os.ReadFile("../../../modules/project/v2/module.seme")
	if err != nil {
		t.Fatal(err)
	}
	set, err := ResolveProjectContractSetV2(e, p, v2)
	if err != nil {
		t.Fatal(err)
	}
	if !set.Validated() || set.Project().Pin().Revision != projectRevV2 || len(set.Project().Exports()) != 30 {
		t.Fatal("unexpected v2 contract set")
	}
	if _, err := ResolveProjectContractSet(e, p, v2); err == nil {
		t.Fatal("v1 resolver accepted v2 substitution")
	}
}

func TestResolveProjectContractSetV3(t *testing.T) {
	e, _, _ := artifacts(t)
	p, err := os.ReadFile("../../../modules/package/v2/module.seme")
	if err != nil {
		t.Fatal(err)
	}
	r, err := os.ReadFile("../../../modules/project/v3/module.seme")
	if err != nil {
		t.Fatal(err)
	}
	set, err := ResolveProjectContractSetV3(e, p, r)
	if err != nil {
		t.Fatal(err)
	}
	if !set.Validated() || set.Package().Pin().Revision != packageRevV2 || set.Project().Pin().Revision != projectRevV3 || len(set.Package().Exports()) != 61 || len(set.Project().Exports()) != 38 {
		t.Fatal("unexpected v3 contract set")
	}
	if _, err = ResolveProjectContractSetV2(e, p, r); err == nil {
		t.Fatal("v2 resolver accepted v3 substitutions")
	}
}

func TestResolveProjectContractSetV4ExactImmutablePins(t *testing.T) {
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	e := read("../../../modules/execution/v35/module.seme")
	p := read("../../../modules/package/v2/module.seme")
	d := read("../../../modules/dependency/v1/module.seme")
	r := read("../../../modules/project/v4/module.seme")
	set, err := ResolveProjectContractSetV4(e, p, d, r)
	if err != nil {
		t.Fatal(err)
	}
	if !set.Validated() || set.Execution().Pin() != (Pin{executionModule, executionRev}) || set.Package().Pin() != (Pin{packageModule, packageRevV2}) || set.Dependency().Pin() != (Pin{dependencyModule, dependencyRev}) || set.Project().Pin() != (Pin{projectModule, projectRevV4}) {
		t.Fatal("unexpected v4 contract pins")
	}
	copy := set.Dependency().Envelope()
	delete(copy.Entities, copy.Module)
	if _, ok := set.Dependency().Envelope().Entities[dependencyModule]; !ok {
		t.Fatal("mutable dependency contract escaped")
	}
	if (ProjectContractSetV4{}).Validated() {
		t.Fatal("zero set authenticated")
	}
	if _, err = ResolveProjectContractSetV4(e, read("../../../modules/package/v1/module.seme"), d, r); err == nil {
		t.Fatal("mixed v1 Package pin accepted")
	}
	if _, err = ResolveProjectContractSetV4(e, p, d, read("../../../modules/project/v3/module.seme")); err == nil {
		t.Fatal("mixed v3 Project pin accepted")
	}
	badDependency, _ := wire.Decode(d)
	badDependency.Revision = mustID("0000000000000000000000000000f002")
	badD, _ := wire.Encode(badDependency)
	if _, err = ResolveProjectContractSetV4(e, p, badD, r); err == nil {
		t.Fatal("wrong Dependency revision accepted")
	}
	badProject, _ := wire.Decode(r)
	m := badProject.Entities[badProject.Module]
	for _, v := range m.Fields[fImports].List {
		q := badProject.Entities[v.Reference]
		if q.Fields[fImportMod].Reference == dependencyModule {
			x := q.Fields[fImportRev]
			x.Bytes = append([]byte(nil), x.Bytes...)
			x.Bytes[15] ^= 1
			q.Fields[fImportRev] = x
			badProject.Entities[q.ID] = q
			break
		}
	}
	badR, _ := wire.Encode(badProject)
	if _, err = ResolveProjectContractSetV4(e, p, d, badR); err == nil {
		t.Fatal("wrong Project dependency pin accepted")
	}
}

func TestResolveProjectContractSetV5ExactImmutablePins(t *testing.T) {
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	e := read("../../../modules/execution/v35/module.seme")
	p := read("../../../modules/package/v3/module.seme")
	d := read("../../../modules/dependency/v1/module.seme")
	r := read("../../../modules/project/v5/module.seme")
	set, err := ResolveProjectContractSetV5(e, p, d, r)
	if err != nil {
		t.Fatal(err)
	}
	if !set.Validated() || set.Package().Pin() != (Pin{packageModule, packageRevV3}) || set.Project().Pin() != (Pin{projectModule, projectRevV5}) {
		t.Fatal("unexpected v5 pins")
	}
	if (ProjectContractSetV5{}).Validated() {
		t.Fatal("zero set authenticated")
	}
	if _, err = ResolveProjectContractSetV5(e, read("../../../modules/package/v2/module.seme"), d, r); err == nil {
		t.Fatal("v2 package substitution accepted")
	}
	if _, err = ResolveProjectContractSetV5(e, p, d, read("../../../modules/project/v4/module.seme")); err == nil {
		t.Fatal("v4 project substitution accepted")
	}
	bad, _ := wire.Decode(r)
	bad.Revision = projectRevV4
	encoded, _ := wire.Encode(bad)
	if _, err = ResolveProjectContractSetV5(e, p, d, encoded); err == nil {
		t.Fatal("wrong revision accepted")
	}
}

func TestResolveProjectContractSetV6AuthenticatesConfiguration(t *testing.T) {
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	f := read("../../../modules/foundation/v1/module.seme")
	e := read("../../../modules/execution/v35/module.seme")
	p := read("../../../modules/package/v3/module.seme")
	d := read("../../../modules/dependency/v1/module.seme")
	c := read("../../../modules/configuration/v1/module.seme")
	r := read("../../../modules/project/v6/module.seme")
	set, err := ResolveProjectContractSetV6(f, e, p, d, c, r)
	if err != nil {
		t.Fatal(err)
	}
	if !set.Validated() || set.Configuration().Pin() != (Pin{configurationModule, configurationRev}) || set.Project().Pin() != (Pin{projectModule, projectRevV6}) {
		t.Fatal("wrong v6 pins")
	}
	bad := append([]byte(nil), c...)
	bad[len(bad)-1] ^= 1
	if got, err := ResolveProjectContractSetV6(f, e, p, d, bad, r); err == nil || got.Validated() {
		t.Fatal("tampered configuration accepted")
	}
	if got, err := ResolveProjectContractSetV6(f, e, p, d, c, read("../../../modules/project/v5/module.seme")); err == nil || got.Validated() {
		t.Fatal("wrong Project accepted")
	}
}

func TestResolveProjectContractSetV7AuthenticatesBoundConfiguration(t *testing.T) {
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	f := read("../../../modules/foundation/v1/module.seme")
	e := read("../../../modules/execution/v35/module.seme")
	p := read("../../../modules/package/v3/module.seme")
	d := read("../../../modules/dependency/v1/module.seme")
	c := read("../../../modules/configuration/v2/module.seme")
	r := read("../../../modules/project/v7/module.seme")
	set, err := ResolveProjectContractSetV7(f, e, p, d, c, r)
	if err != nil {
		t.Fatal(err)
	}
	if !set.Validated() || set.Configuration().Pin() != (Pin{configurationModule, configurationRevV2}) || set.Project().Pin() != (Pin{projectModule, projectRevV7}) {
		t.Fatal("wrong v7 pins")
	}
	if got, err := ResolveProjectContractSetV7(f, e, p, d, read("../../../modules/configuration/v1/module.seme"), r); err == nil || got.Validated() {
		t.Fatal("v1 configuration substitution accepted")
	}
	if got, err := ResolveProjectContractSetV7(f, e, p, d, c, read("../../../modules/project/v6/module.seme")); err == nil || got.Validated() {
		t.Fatal("v6 project substitution accepted")
	}
}

func TestResolveProjectContractSetV8AuthenticatesOneExecutionV36Snapshot(t *testing.T) {
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	f := read("../../../modules/foundation/v1/module.seme")
	e := read("../../../modules/execution/v36/module.seme")
	p := read("../../../modules/package/v4/module.seme")
	d := read("../../../modules/dependency/v1/module.seme")
	c := read("../../../modules/configuration/v3/module.seme")
	r := read("../../../modules/project/v8/module.seme")
	set, err := ResolveProjectContractSetV8(f, e, p, d, c, r)
	if err != nil || !set.Validated() || set.Execution().Pin().Revision != executionRevV36 || set.Package().Pin().Revision != packageRevV4 || set.Configuration().Pin().Revision != configurationRevV3 || set.Project().Pin().Revision != projectRevV8 {
		t.Fatalf("resolve v8: validated=%v err=%v", set.Validated(), err)
	}
	if bad, err := ResolveProjectContractSetV8(f, read("../../../modules/execution/v35/module.seme"), p, d, c, r); err == nil || bad.Validated() {
		t.Fatal("v8 accepted Execution v35")
	}
	if bad, err := ResolveProjectContractSetV8(f, e, read("../../../modules/package/v3/module.seme"), d, c, r); err == nil || bad.Validated() {
		t.Fatal("v8 accepted Package v3")
	}
	if bad, err := ResolveProjectContractSetV8(f, e, p, d, read("../../../modules/configuration/v2/module.seme"), r); err == nil || bad.Validated() {
		t.Fatal("v8 accepted Configuration v2")
	}
	if bad, err := ResolveProjectContractSetV8(f, e, p, d, c, read("../../../modules/project/v7/module.seme")); err == nil || bad.Validated() {
		t.Fatal("v8 accepted Project v7")
	}
}

func TestResolveProjectContractSetV9AuthenticatesResources(t *testing.T) {
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	f := read("../../../modules/foundation/v1/module.seme")
	e := read("../../../modules/execution/v36/module.seme")
	p := read("../../../modules/package/v4/module.seme")
	d := read("../../../modules/dependency/v1/module.seme")
	c := read("../../../modules/configuration/v3/module.seme")
	q := read("../../../modules/resource/v1/module.seme")
	r := read("../../../modules/project/v9/module.seme")
	set, err := ResolveProjectContractSetV9(f, e, p, d, c, q, r)
	if err != nil || !set.Validated() || !set.Resource().Validated() {
		t.Fatalf("v9: %v", err)
	}
	if bad, err := ResolveProjectContractSetV9(f, e, p, d, c, read("../../../modules/dependency/v1/module.seme"), r); err == nil || bad.Validated() {
		t.Fatal("accepted resource substitution")
	}
	if bad, err := ResolveProjectContractSetV9(f, e, p, d, c, q, read("../../../modules/project/v8/module.seme")); err == nil || bad.Validated() {
		t.Fatal("accepted project v8")
	}
}

func TestResolveProjectContractSetV3RejectsMutations(t *testing.T) {
	e, _, _ := artifacts(t)
	p, _ := os.ReadFile("../../../modules/package/v2/module.seme")
	r, _ := os.ReadFile("../../../modules/project/v3/module.seme")
	mutate := func(source []byte, f func(*wire.Envelope)) []byte {
		x, err := wire.Decode(source)
		if err != nil {
			t.Fatal(err)
		}
		f(&x)
		out, err := wire.Encode(x)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	tests := map[string]func() error{
		"package-revision": func() error {
			bad := mutate(p, func(x *wire.Envelope) { x.Revision = packageRev })
			_, err := ResolveProjectContractSetV3(e, bad, r)
			return err
		},
		"project-revision": func() error {
			bad := mutate(r, func(x *wire.Envelope) { x.Revision = projectRevV2 })
			_, err := ResolveProjectContractSetV3(e, p, bad)
			return err
		},
		"package-export": func() error {
			bad := mutate(p, func(x *wire.Envelope) {
				m := x.Entities[x.Module]
				v := m.Fields[fExports]
				v.List = v.List[1:]
				m.Fields[fExports] = v
				x.Entities[x.Module] = m
			})
			_, err := ResolveProjectContractSetV3(e, bad, r)
			return err
		},
		"project-export": func() error {
			bad := mutate(r, func(x *wire.Envelope) {
				m := x.Entities[x.Module]
				v := m.Fields[fExports]
				v.List = v.List[1:]
				m.Fields[fExports] = v
				x.Entities[x.Module] = m
			})
			_, err := ResolveProjectContractSetV3(e, p, bad)
			return err
		},
		"project-import-pin": func() error {
			bad := mutate(r, func(x *wire.Envelope) {
				m := x.Entities[x.Module]
				iid := m.Fields[fImports].List[0].Reference
				im := x.Entities[iid]
				v := im.Fields[fImportRev]
				v.Bytes = append([]byte(nil), v.Bytes...)
				v.Bytes[15] ^= 1
				im.Fields[fImportRev] = v
				x.Entities[iid] = im
			})
			_, err := ResolveProjectContractSetV3(e, p, bad)
			return err
		},
	}
	for name, run := range tests {
		t.Run(name, func(t *testing.T) {
			if err := run(); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	t.Run("package-digest", func(t *testing.T) {
		if _, err := Resolve(p, Expectation{Pin: Pin{packageModule, packageRevV2}, ModuleVersion: 2, Digest: mustDigest("0100000000000000000000000000000000000000000000000000000000000000")}); err == nil || !strings.Contains(err.Error(), "digest") {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("project-digest", func(t *testing.T) {
		if _, err := Resolve(r, Expectation{Pin: Pin{projectModule, projectRevV3}, ModuleVersion: 3, Digest: mustDigest("0100000000000000000000000000000000000000000000000000000000000000")}); err == nil || !strings.Contains(err.Error(), "digest") {
			t.Fatalf("got %v", err)
		}
	})
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

func TestResolveProjectContractSetV10ExactPins(t *testing.T) {
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	s, err := ResolveProjectContractSetV10(read("../../../modules/foundation/v1/module.seme"), read("../../../modules/execution/v36/module.seme"), read("../../../modules/package/v4/module.seme"), read("../../../modules/dependency/v1/module.seme"), read("../../../modules/configuration/v3/module.seme"), read("../../../modules/resource/v1/module.seme"), read("../../../modules/durable-state/v1/module.seme"), read("../../../modules/project/v9/module.seme"), read("../../../modules/project/v10/module.seme"))
	if err != nil || !s.Validated() {
		t.Fatalf("resolve: %v", err)
	}
	if s.DurableState().Pin() != (Pin{durableStateModule, durableStateRev}) || s.Project().Pin() != (Pin{projectModule, projectRevV10}) {
		t.Fatal("pins")
	}
	if _, err = ResolveProjectContractSetV10(read("../../../modules/foundation/v1/module.seme"), read("../../../modules/execution/v36/module.seme"), read("../../../modules/package/v4/module.seme"), read("../../../modules/dependency/v1/module.seme"), read("../../../modules/configuration/v3/module.seme"), read("../../../modules/resource/v1/module.seme"), read("../../../modules/resource/v1/module.seme"), read("../../../modules/project/v9/module.seme"), read("../../../modules/project/v10/module.seme")); err == nil {
		t.Fatal("accepted substituted durable contract")
	}
}
