package packagedetailemitter

import (
	"bytes"
	"crypto/sha256"
	"os"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/packagedetail"
	"seme.local/reference/projectemitter"
	"seme.local/reference/wire"
)

func TestEmitDeterministicPrivatePublicAndLocalImport(t *testing.T) {
	base, graph := fixture(t)
	before, _ := wire.Encode(base)
	a, err := Emit(base, graph)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Emit(base, graph)
	if err != nil || !bytes.Equal(a, b) {
		t.Fatalf("repeat differs: %v", err)
	}
	after, _ := wire.Encode(base)
	if !bytes.Equal(before, after) {
		t.Fatal("base mutated")
	}
	if packagePin(base) != id("b001") {
		t.Fatal("input pin changed")
	}
	decoded, _ := wire.Decode(a)
	if packagePin(decoded) != id("b002") {
		t.Fatal("output pin was not upgraded")
	}
	origins := 0
	for _, entity := range decoded.Entities {
		if entity.Schema == id("b026") {
			origins++
		}
	}
	if origins != 2 {
		t.Fatalf("shared origins were not reused: %d", origins)
	}
}

func TestEmitRejectsAtomically(t *testing.T) {
	base, graph := fixture(t)
	tests := map[string]func(*wire.Envelope, *packagedetail.Graph){
		"invalid-model": func(_ *wire.Envelope, g *packagedetail.Graph) {
			g.Packages[0].Members[0].Visibility = packagedetail.Package
		},
		"missing-package": func(_ *wire.Envelope, g *packagedetail.Graph) { g.Packages[0].Identity = "missing" },
		"noncanonical-order": func(_ *wire.Envelope, g *packagedetail.Graph) {
			g.Packages[0], g.Packages[1] = g.Packages[1], g.Packages[0]
		},
		"missing-declaration": func(_ *wire.Envelope, g *packagedetail.Graph) {
			g.Packages[0].Members[0].Identity = id("ffff").String()
		},
		"missing-source": func(e *wire.Envelope, _ *packagedetail.Graph) { delete(e.Entities, id("a001")) },
		"bad-external": func(_ *wire.Envelope, g *packagedetail.Graph) {
			g.Packages[1].Imports[0].Class = packagedetail.External
			g.Packages[1].Imports[0].Resolved = id("ffff").String()
		},
		"collision": func(e *wire.Envelope, _ *packagedetail.Graph) {
			x := stable("graph")
			e.Entities[x] = wire.Entity{ID: x, Schema: id("9010"), Version: 1, Fields: map[wire.ID]wire.Value{}}
		},
		"malformed-base": func(e *wire.Envelope, _ *packagedetail.Graph) { e.Entities[id("aaaa")] = wire.Entity{ID: id("bbbb")} },
		"missing-pin":    func(e *wire.Envelope, _ *packagedetail.Graph) { removePackagePins(e) },
		"duplicate-pin": func(e *wire.Envelope, _ *packagedetail.Graph) {
			x := id("afff")
			e.Entities[x] = wire.Entity{ID: x, Schema: id("13"), Version: 1, Fields: map[wire.ID]wire.Value{id("130"): {Tag: 6, Reference: id("b000")}, id("131"): {Tag: 5, Bytes: idBytes(id("b002"))}}}
			m := e.Entities[e.Module]
			v := m.Fields[id("121")]
			v.List = append(v.List, wire.Value{Tag: 6, Reference: x})
			m.Fields[id("121")] = v
			e.Entities[m.ID] = m
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			e := copyEnvelope(base)
			g := copyGraph(graph)
			mutate(&e, &g)
			before, _ := wire.Encode(e)
			out, err := Emit(e, g)
			if err == nil || out != nil {
				t.Fatalf("output=%d err=%v", len(out), err)
			}
			after, _ := wire.Encode(e)
			if !bytes.Equal(before, after) {
				t.Fatal("failed emission mutated input")
			}
		})
	}
}

func fixture(t *testing.T) (wire.Envelope, packagedetail.Graph) {
	t.Helper()
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	contracts, err := contractcatalog.ResolveProjectContractSet(read("../../../modules/execution/v35/module.seme"), read("../../../modules/package/v1/module.seme"), read("../../../modules/project/v1/module.seme"))
	if err != nil {
		t.Fatal(err)
	}
	typ, p1, p2, p3 := id("8001"), id("8002"), id("8003"), id("8004")
	b1, b2, b3 := id("8005"), id("8006"), id("8007")
	f1, f2, f3, program := id("8010"), id("8011"), id("8012"), id("8013")
	r := func(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
	l := func(xs ...wire.Value) wire.Value { return wire.Value{Tag: 7, List: xs} }
	param := func(x wire.ID) wire.Entity {
		return wire.Entity{ID: x, Schema: id("9012"), Version: 1, Fields: map[wire.ID]wire.Value{id("9120"): {Tag: 5, Bytes: []byte("value")}, id("9121"): r(typ), id("9122"): {Tag: 3}}}
	}
	block := func(x, p wire.ID) wire.Entity {
		return wire.Entity{ID: x, Schema: id("9013"), Version: 1, Fields: map[wire.ID]wire.Value{id("9130"): r(p)}}
	}
	fn := func(x, p, b wire.ID, n string) wire.Entity {
		return wire.Entity{ID: x, Schema: id("9011"), Version: 1, Fields: map[wire.ID]wire.Value{id("9110"): {Tag: 5, Bytes: []byte(n)}, id("9111"): l(r(p)), id("9112"): r(typ), id("9113"): r(b)}}
	}
	execution := wire.Envelope{Module: id("9000"), Revision: id("9023"), Entities: map[wire.ID]wire.Entity{
		typ: {ID: typ, Schema: id("9010"), Version: 1, Fields: map[wire.ID]wire.Value{id("9100"): {Tag: 3, Unsigned: 64}, id("9101"): {Tag: 2}, id("9102"): {Tag: 3}}},
		p1:  param(p1), p2: param(p2), p3: param(p3), b1: block(b1, p1), b2: block(b2, p2), b3: block(b3, p3), f1: fn(f1, p1, b1, "Normalize"), f2: fn(f2, p2, b2, "Apply"), f3: fn(f3, p3, b3, "helper"),
		program: {ID: program, Schema: id("9015"), Version: 1, Fields: map[wire.ID]wire.Value{id("9150"): l(r(f1), r(f2), r(f3)), id("9151"): r(f2)}},
	}}
	raw, err := projectemitter.Emit(contracts, projectemitter.Input{Identity: "example.test/tool", RootPackage: "example.test/tool/root", Execution: execution, Packages: []projectemitter.Package{
		{Name: "example.test/tool/model", Interfaces: []projectemitter.Interface{{Name: "Normalize", Function: f1, Parameters: []wire.ID{typ}, Result: typ}}},
		{Name: "example.test/tool/root", Dependencies: []projectemitter.Dependency{{Name: "domain", Package: "example.test/tool/model"}}, Interfaces: []projectemitter.Interface{{Name: "Apply", Function: f2, Parameters: []wire.ID{typ}, Result: typ}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	base, _ := wire.Decode(raw)
	s1, s2 := id("a001"), id("a002")
	base.Entities[s1] = wire.Entity{ID: s1, Schema: id("e015"), Version: 1, Fields: map[wire.ID]wire.Value{}}
	base.Entities[s2] = wire.Entity{ID: s2, Schema: id("e015"), Version: 1, Fields: map[wire.ID]wire.Value{}}
	d1, d2 := sha256.Sum256([]byte("model")), sha256.Sum256([]byte("root"))
	o1 := packagedetail.Origin{SourceIdentity: s1.String(), Path: "model/value.go", ContentDigest: d1, ByteEnd: 10, StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 11}
	o2 := packagedetail.Origin{SourceIdentity: s2.String(), Path: "root/root.go", ContentDigest: d2, ByteEnd: 20, StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 21}
	graph := packagedetail.Graph{Packages: []packagedetail.Detail{
		{Identity: "example.test/tool/model", Sources: []packagedetail.Source{{Identity: s1.String(), Path: o1.Path, ContentDigest: d1, ByteSize: 10}}, Members: []packagedetail.Member{{Identity: f1.String(), Name: "Normalize", ExportName: "Normalize", Visibility: packagedetail.Public, Origin: o1, Callable: true, Parameters: []string{typ.String()}, Results: []string{typ.String()}}}},
		{Identity: "example.test/tool/root", Root: true, Sources: []packagedetail.Source{{Identity: s2.String(), Path: o2.Path, ContentDigest: d2, ByteSize: 20}}, Members: []packagedetail.Member{{Identity: f2.String(), Name: "Apply", ExportName: "Apply", Visibility: packagedetail.Public, Origin: o2, Callable: true, Parameters: []string{typ.String()}, Results: []string{typ.String()}}, {Identity: f3.String(), Name: "helper", Visibility: packagedetail.Package, Origin: o2, Callable: true, Parameters: []string{typ.String()}, Results: []string{typ.String()}}}, Imports: []packagedetail.Import{{Alias: "modelAlias", Requested: "domain", Resolved: "example.test/tool/model", Class: packagedetail.Local, Origin: o2}}},
	}}
	return base, graph
}

func copyEnvelope(e wire.Envelope) wire.Envelope {
	raw, _ := wire.Encode(e)
	out, _ := wire.Decode(raw)
	return out
}
func copyGraph(g packagedetail.Graph) packagedetail.Graph {
	out := packagedetail.Graph{Packages: make([]packagedetail.Detail, len(g.Packages))}
	for i, p := range g.Packages {
		out.Packages[i] = p
		out.Packages[i].Sources = append([]packagedetail.Source(nil), p.Sources...)
		out.Packages[i].Members = append([]packagedetail.Member(nil), p.Members...)
		out.Packages[i].Imports = append([]packagedetail.Import(nil), p.Imports...)
	}
	return out
}
func idBytes(x wire.ID) []byte { return append([]byte(nil), x[:]...) }
func packagePin(e wire.Envelope) wire.ID {
	var out wire.ID
	count := 0
	for _, q := range e.Entities {
		if q.Schema == id("13") && q.Fields[id("130")].Tag == 6 && q.Fields[id("130")].Reference == id("b000") {
			b := q.Fields[id("131")].Bytes
			if len(b) == 16 {
				copy(out[:], b)
			}
			count++
		}
	}
	if count != 1 {
		return wire.ID{}
	}
	return out
}
func removePackagePins(e *wire.Envelope) {
	m := e.Entities[e.Module]
	v := m.Fields[id("121")]
	kept := v.List[:0]
	for _, x := range v.List {
		q := e.Entities[x.Reference]
		if q.Schema == id("13") && q.Fields[id("130")].Reference == id("b000") {
			delete(e.Entities, x.Reference)
		} else {
			kept = append(kept, x)
		}
	}
	v.List = kept
	m.Fields[id("121")] = v
	e.Entities[m.ID] = m
}
