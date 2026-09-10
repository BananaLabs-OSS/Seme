package packagedetailinstance

import (
	"crypto/sha256"
	"os"
	"sort"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectemitter"
	"seme.local/reference/wire"
)

func TestValidateCompletePackageDetailGraph(t *testing.T) {
	source := validArtifact(t)
	if err := Validate(source); err != nil {
		t.Fatal(err)
	}
	e, _ := wire.Decode(source)
	d := idsWithSchema(e, detailSchema)[0]
	a, _ := DetailRevision(e, d)
	b, _ := DetailRevision(e, d)
	if string(a) != string(b) {
		t.Fatal("detail revision is not deterministic")
	}
	a, _ = ContentRevision(e, idsWithSchema(e, graphSchema)[0])
	b, _ = ContentRevision(e, idsWithSchema(e, graphSchema)[0])
	if string(a) != string(b) {
		t.Fatal("content revision is not deterministic")
	}
}

func TestValidateAcceptsV2PinWithoutRedundantV1Pin(t *testing.T) {
	e, _ := wire.Decode(validArtifact(t))
	m := e.Entities[e.Module]
	imports := m.Fields[id("121")]
	kept := imports.List[:0]
	for _, value := range imports.List {
		q := e.Entities[value.Reference]
		if q.Schema == importSchema && string(q.Fields[id("131")].Bytes) == string(idBytes(id("b001"))) {
			delete(e.Entities, value.Reference)
			continue
		}
		kept = append(kept, value)
	}
	imports.List = kept
	m.Fields[id("121")] = imports
	e.Entities[m.ID] = m
	raw, err := wire.Encode(e)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(raw); err != nil {
		t.Fatalf("v2-only pin rejected: %v", err)
	}
}

func TestRejectsAdversarialPackageDetailGraphs(t *testing.T) {
	base, _ := wire.Decode(validArtifact(t))
	tests := map[string]func(*wire.Envelope){
		"graph-count": func(e *wire.Envelope) {
			g := e.Entities[idsWithSchema(*e, graphSchema)[0]]
			g.ID = id("f001")
			e.Entities[g.ID] = g
		},
		"package-coverage": func(e *wire.Envelope) {
			g := e.Entities[idsWithSchema(*e, graphSchema)[0]]
			g.Fields[id("b200")] = wire.Value{Tag: 7, List: g.Fields[id("b200")].List[:1]}
			e.Entities[g.ID] = g
		},
		"visibility-enum": func(e *wire.Envelope) {
			x := idsWithSchema(*e, visibilitySchema)[0]
			q := e.Entities[x]
			q.Fields[id("b230")] = wire.Value{Tag: 3, Unsigned: 3}
			e.Entities[x] = q
		},
		"visible-without-export": func(e *wire.Envelope) {
			x := idsWithSchema(*e, memberSchema)[0]
			q := e.Entities[x]
			delete(q.Fields, id("b223"))
			e.Entities[x] = q
		},
		"origin-digest": func(e *wire.Envelope) {
			x := idsWithSchema(*e, originSchema)[0]
			q := e.Entities[x]
			q.Fields[id("b262")] = wire.Value{Tag: 5, Bytes: []byte("short")}
			e.Entities[x] = q
		},
		"origin-range": func(e *wire.Envelope) {
			x := idsWithSchema(*e, originSchema)[0]
			q := e.Entities[x]
			q.Fields[id("b263")] = wire.Value{Tag: 3, Unsigned: 9}
			q.Fields[id("b264")] = wire.Value{Tag: 3, Unsigned: 2}
			e.Entities[x] = q
		},
		"origin-position": func(e *wire.Envelope) {
			x := idsWithSchema(*e, originSchema)[0]
			q := e.Entities[x]
			q.Fields[id("b267")] = q.Fields[id("b265")]
			q.Fields[id("b268")] = q.Fields[id("b266")]
			e.Entities[x] = q
		},
		"origin-owned-twice": func(e *wire.Envelope) {
			details := idsWithSchema(*e, detailSchema)
			first := e.Entities[details[0]].Fields[id("b213")].List[0].Reference
			q := e.Entities[details[1]]
			q.Fields[id("b213")] = wire.Value{Tag: 7, List: []wire.Value{{Tag: 6, Reference: first}}}
			e.Entities[q.ID] = q
		},
		"declaration-owned-twice": func(e *wire.Envelope) {
			members := idsWithSchema(*e, memberSchema)
			first := e.Entities[members[0]].Fields[id("b220")]
			q := e.Entities[members[1]]
			q.Fields[id("b220")] = first
			e.Entities[q.ID] = q
		},
		"program-function-unowned": func(e *wire.Envelope) {
			program := idsWithSchema(*e, id("9015"))[0]
			q := e.Entities[program]
			original := q.Fields[id("9150")].List[0].Reference
			private := e.Entities[original]
			private.ID = id("f010")
			private.Fields[id("9110")] = wire.Value{Tag: 5, Bytes: []byte("private")}
			e.Entities[private.ID] = private
			members := q.Fields[id("9150")]
			members.List = append(members.List, wire.Value{Tag: 6, Reference: private.ID})
			q.Fields[id("9150")] = members
			e.Entities[q.ID] = q
		},
		"export-interface": func(e *wire.Envelope) {
			x := idsWithSchema(*e, memberSchema)[0]
			q := e.Entities[x]
			q.Fields[id("b223")] = wire.Value{Tag: 5, Bytes: []byte("Forged")}
			e.Entities[x] = q
		},
		"local-arm": func(e *wire.Envelope) {
			x := idsWithSchema(*e, bindingSchema)[0]
			q := e.Entities[x]
			delete(q.Fields, id("b243"))
			e.Entities[x] = q
		},
		"orphan": func(e *wire.Envelope) {
			x := idsWithSchema(*e, visibilitySchema)[0]
			q := e.Entities[x]
			q.ID = id("f002")
			e.Entities[q.ID] = q
		},
		"detail-revision": func(e *wire.Envelope) {
			x := idsWithSchema(*e, detailSchema)[0]
			q := e.Entities[x]
			q.Fields[id("b214")] = wire.Value{Tag: 5, Bytes: make([]byte, 32)}
			e.Entities[x] = q
		},
		"content-revision": func(e *wire.Envelope) {
			x := idsWithSchema(*e, graphSchema)[0]
			q := e.Entities[x]
			q.Fields[id("b201")] = wire.Value{Tag: 5, Bytes: make([]byte, 32)}
			e.Entities[x] = q
		},
		"missing-v2-pin": func(e *wire.Envelope) {
			m := e.Entities[e.Module]
			var list []wire.Value
			for _, v := range m.Fields[id("121")].List {
				q := e.Entities[v.Reference]
				b := q.Fields[id("131")].Bytes
				if string(b) != string(idBytes(id("b002"))) {
					list = append(list, v)
				} else {
					delete(e.Entities, v.Reference)
				}
			}
			m.Fields[id("121")] = wire.Value{Tag: 7, List: list}
			e.Entities[m.ID] = m
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			e := cloneEnvelope(base)
			mutate(&e)
			raw, err := wire.Encode(e)
			if err != nil {
				t.Fatal(err)
			}
			if Validate(raw) == nil {
				t.Fatal("accepted adversarial graph")
			}
		})
	}
}

func validArtifact(t *testing.T) []byte {
	t.Helper()
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	contracts, e := contractcatalog.ResolveProjectContractSet(read("../../../modules/execution/v35/module.seme"), read("../../../modules/package/v1/module.seme"), read("../../../modules/project/v1/module.seme"))
	if e != nil {
		t.Fatal(e)
	}
	typ, p1, p2, b1, b2, f1, f2, program := id("8001"), id("8002"), id("8003"), id("8004"), id("8005"), id("8006"), id("8007"), id("8008")
	ref := func(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
	list := func(xs ...wire.Value) wire.Value { return wire.Value{Tag: 7, List: xs} }
	param := func(x wire.ID) wire.Entity {
		return wire.Entity{ID: x, Schema: id("9012"), Version: 1, Fields: map[wire.ID]wire.Value{id("9120"): {Tag: 5, Bytes: []byte("value")}, id("9121"): ref(typ), id("9122"): {Tag: 3}}}
	}
	fn := func(x, p, b wire.ID, n string) wire.Entity {
		return wire.Entity{ID: x, Schema: id("9011"), Version: 1, Fields: map[wire.ID]wire.Value{id("9110"): {Tag: 5, Bytes: []byte(n)}, id("9111"): list(ref(p)), id("9112"): ref(typ), id("9113"): ref(b)}}
	}
	ex := wire.Envelope{Module: id("9000"), Revision: id("9023"), Entities: map[wire.ID]wire.Entity{typ: {ID: typ, Schema: id("9010"), Version: 1, Fields: map[wire.ID]wire.Value{id("9100"): {Tag: 3, Unsigned: 64}, id("9101"): {Tag: 2}, id("9102"): {Tag: 3}}}, p1: param(p1), p2: param(p2), b1: {ID: b1, Schema: id("9013"), Version: 1, Fields: map[wire.ID]wire.Value{id("9130"): ref(p1)}}, b2: {ID: b2, Schema: id("9013"), Version: 1, Fields: map[wire.ID]wire.Value{id("9130"): ref(p2)}}, f1: fn(f1, p1, b1, "Normalize"), f2: fn(f2, p2, b2, "Apply"), program: {ID: program, Schema: id("9015"), Version: 1, Fields: map[wire.ID]wire.Value{id("9150"): list(ref(f1), ref(f2)), id("9151"): ref(f2)}}}}
	raw, e := projectemitter.Emit(contracts, projectemitter.Input{
		Identity: "example.test/tool", RootPackage: "example.test/tool/root", Execution: ex,
		Packages: []projectemitter.Package{
			{Name: "example.test/tool/model", Interfaces: []projectemitter.Interface{{Name: "Normalize", Function: f1, Parameters: []wire.ID{typ}, Result: typ}}},
			{Name: "example.test/tool/root", Dependencies: []projectemitter.Dependency{{Name: "model", Package: "example.test/tool/model"}}, Interfaces: []projectemitter.Interface{{Name: "Apply", Function: f2, Parameters: []wire.ID{typ}, Result: typ}}},
		},
	})
	if e != nil {
		t.Fatal(e)
	}
	env, _ := wire.Decode(raw)
	var model, root wire.ID
	for x, q := range env.Entities {
		if q.Schema == packageSchema {
			n := string(q.Fields[id("b100")].Bytes)
			if n == "example.test/tool/model" {
				model = x
			} else {
				root = x
			}
		}
	}
	source := id("a001")
	env.Entities[source] = wire.Entity{ID: source, Schema: id("e015"), Version: 1, Fields: map[wire.ID]wire.Value{}}
	origin := func(x wire.ID, p string) wire.Entity {
		h := sha256.Sum256([]byte(p))
		return wire.Entity{ID: x, Schema: originSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("b260"): ref(source), id("b261"): {Tag: 5, Bytes: []byte(p)}, id("b262"): {Tag: 5, Bytes: h[:]}, id("b263"): {Tag: 3}, id("b264"): {Tag: 3, Unsigned: 10}, id("b265"): {Tag: 3, Unsigned: 1}, id("b266"): {Tag: 3, Unsigned: 1}, id("b267"): {Tag: 3, Unsigned: 1}, id("b268"): {Tag: 3, Unsigned: 11}}}
	}
	o1, o2, v1, v2, c, m1, m2, binding, d1, d2, g := id("a010"), id("a011"), id("a012"), id("a013"), id("a014"), id("a015"), id("a016"), id("a017"), id("a018"), id("a019"), id("a020")
	env.Entities[o1] = origin(o1, "model/value.go")
	env.Entities[o2] = origin(o2, "root/root.go")
	env.Entities[v1] = wire.Entity{ID: v1, Schema: visibilitySchema, Version: 1, Fields: map[wire.ID]wire.Value{id("b230"): {Tag: 3, Unsigned: 2}}}
	env.Entities[v2] = wire.Entity{ID: v2, Schema: visibilitySchema, Version: 1, Fields: map[wire.ID]wire.Value{id("b230"): {Tag: 3, Unsigned: 2}}}
	env.Entities[c] = wire.Entity{ID: c, Schema: classSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("b250"): {Tag: 3}}}
	member := func(x, decl, vis, ori wire.ID, n string) wire.Entity {
		return wire.Entity{ID: x, Schema: memberSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("b220"): ref(decl), id("b221"): {Tag: 5, Bytes: []byte(n)}, id("b222"): ref(vis), id("b223"): {Tag: 5, Bytes: []byte(n)}, id("b224"): ref(ori)}}
	}
	env.Entities[m1] = member(m1, f1, v1, o1, "Normalize")
	env.Entities[m2] = member(m2, f2, v2, o2, "Apply")
	env.Entities[binding] = wire.Entity{ID: binding, Schema: bindingSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("b240"): {Tag: 5, Bytes: []byte("model")}, id("b241"): {Tag: 5, Bytes: []byte("example.test/tool/model")}, id("b242"): ref(c), id("b243"): ref(model), id("b245"): ref(o2)}}
	detail := func(x, p, m, o wire.ID, imports []wire.Value) wire.Entity {
		return wire.Entity{ID: x, Schema: detailSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("b210"): ref(p), id("b211"): list(ref(m)), id("b212"): list(imports...), id("b213"): list(ref(o)), id("b214"): {Tag: 5, Bytes: make([]byte, 32)}}}
	}
	env.Entities[d1] = detail(d1, model, m1, o1, nil)
	env.Entities[d2] = detail(d2, root, m2, o2, []wire.Value{ref(binding)})
	for _, x := range []wire.ID{d1, d2} {
		q := env.Entities[x]
		r, _ := DetailRevision(env, x)
		q.Fields[id("b214")] = wire.Value{Tag: 5, Bytes: r}
		env.Entities[x] = q
	}
	env.Entities[g] = wire.Entity{ID: g, Schema: graphSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("b200"): list(ref(d1), ref(d2)), id("b201"): {Tag: 5, Bytes: make([]byte, 32)}}}
	q := env.Entities[g]
	r, _ := ContentRevision(env, g)
	q.Fields[id("b201")] = wire.Value{Tag: 5, Bytes: r}
	env.Entities[g] = q
	imp := id("a021")
	env.Entities[imp] = wire.Entity{ID: imp, Schema: importSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("130"): ref(id("b000")), id("131"): {Tag: 5, Bytes: idBytes(id("b002"))}}}
	mod := env.Entities[env.Module]
	imports := mod.Fields[id("121")]
	imports.List = append(imports.List, ref(imp))
	sort.Slice(imports.List, func(i, j int) bool {
		return imports.List[i].Reference.String() < imports.List[j].Reference.String()
	})
	mod.Fields[id("121")] = imports
	env.Entities[mod.ID] = mod
	out, e := wire.Encode(env)
	if e != nil {
		t.Fatal(e)
	}
	return out
}
func idBytes(x wire.ID) []byte { return append([]byte(nil), x[:]...) }
func cloneEnvelope(e wire.Envelope) wire.Envelope {
	raw, _ := wire.Encode(e)
	out, _ := wire.Decode(raw)
	return out
}
