package packageinstance

import (
	"seme.local/reference/wire"
	"strings"
	"testing"
)

func rv(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
func lv(xs ...wire.ID) wire.Value {
	v := wire.Value{Tag: 7}
	for _, x := range xs {
		v.List = append(v.List, rv(x))
	}
	return v
}
func bv(s string) wire.Value { return wire.Value{Tag: 5, Bytes: []byte(s)} }
func ent(x, s wire.ID, fields map[wire.ID]wire.Value) wire.Entity {
	return wire.Entity{ID: x, Schema: s, Version: 1, Fields: fields}
}

func valid() wire.Envelope {
	m := id("10000000000000000000000000000000")
	imp := id("10000000000000000000000000000001")
	pkg := id("20000000000000000000000000000000")
	iface := id("30000000000000000000000000000000")
	fn := id("40000000000000000000000000000000")
	param := id("50000000000000000000000000000000")
	typ := id("60000000000000000000000000000000")
	e := wire.Envelope{Module: m, Revision: id("10000000000000000000000000000002"), Entities: map[wire.ID]wire.Entity{}}
	e.Entities[m] = ent(m, moduleSchema, map[wire.ID]wire.Value{fImports: lv(imp)})
	e.Entities[imp] = ent(imp, importSchema, map[wire.ID]wire.Value{fImportModule: rv(packageModule), fImportRevision: {Tag: 5, Bytes: packageRevision[:]}})
	e.Entities[typ] = ent(typ, id("70000000000000000000000000000000"), map[wire.ID]wire.Value{})
	e.Entities[param] = ent(param, parameterSchema, map[wire.ID]wire.Value{fid(0x9120): bv("value"), fid(0x9121): rv(typ), fid(0x9122): {Tag: 4}})
	e.Entities[fn] = ent(fn, functionSchema, map[wire.ID]wire.Value{fid(0x9110): bv("Apply"), fid(0x9111): lv(param), fid(0x9112): rv(typ), fid(0x9113): rv(typ)})
	e.Entities[iface] = ent(iface, interfaceSchema, map[wire.ID]wire.Value{fid(0xb110): bv("Apply"), fid(0xb111): rv(fn), fid(0xb112): lv(typ), fid(0xb113): rv(typ)})
	e.Entities[pkg] = ent(pkg, packageSchema, map[wire.ID]wire.Value{fid(0xb100): bv("example/app"), fid(0xb101): bv("pending"), fid(0xb102): lv(iface), fid(0xb103): lv(), fid(0xb104): lv(), fid(0xb105): lv(), fid(0xb106): lv()})
	rev, err := Revision(e, pkg)
	if err != nil {
		panic(err)
	}
	q := e.Entities[pkg]
	q.Fields[fid(0xb101)] = wire.Value{Tag: 5, Bytes: rev}
	e.Entities[pkg] = q
	return e
}

func TestValidate(t *testing.T) {
	e := valid()
	if err := ValidateEnvelope(e); err != nil {
		t.Fatal(err)
	}
	a, _ := Revision(e, id("20000000000000000000000000000000"))
	b, _ := Revision(e, id("20000000000000000000000000000000"))
	if string(a) != string(b) {
		t.Fatal("revision is not deterministic")
	}
}

func TestRejectsAdversaries(t *testing.T) {
	tests := map[string]func(*wire.Envelope){
		"unpinned": func(e *wire.Envelope) {
			q := e.Entities[id("10000000000000000000000000000001")]
			q.Fields[fImportRevision] = bv("bad")
			e.Entities[q.ID] = q
		},
		"field-kind": func(e *wire.Envelope) {
			p := id("20000000000000000000000000000000")
			q := e.Entities[p]
			q.Fields[fid(0xb100)] = wire.Value{Tag: 3}
			e.Entities[p] = q
		},
		"missing-target": func(e *wire.Envelope) {
			p := id("20000000000000000000000000000000")
			q := e.Entities[p]
			q.Fields[fid(0xb102)] = lv(id("ffffffffffffffffffffffffffffffff"))
			e.Entities[p] = q
		},
		"signature": func(e *wire.Envelope) {
			x := id("30000000000000000000000000000000")
			q := e.Entities[x]
			q.Fields[fid(0xb112)] = lv()
			e.Entities[x] = q
		},
		"duplicate-interface": func(e *wire.Envelope) {
			p := id("20000000000000000000000000000000")
			i := id("30000000000000000000000000000000")
			q := e.Entities[p]
			q.Fields[fid(0xb102)] = lv(i, i)
			e.Entities[p] = q
		},
		"interface-version": func(e *wire.Envelope) {
			x := id("30000000000000000000000000000000")
			q := e.Entities[x]
			q.Version = 2
			e.Entities[x] = q
		},
		"revision": func(e *wire.Envelope) {
			p := id("20000000000000000000000000000000")
			q := e.Entities[p]
			q.Fields[fid(0xb101)] = bv("forged")
			e.Entities[p] = q
		},
		"package-version": func(e *wire.Envelope) {
			p := id("20000000000000000000000000000000")
			q := e.Entities[p]
			q.Version = 2
			e.Entities[p] = q
		},
		"dependency-missing": func(e *wire.Envelope) {
			p := id("20000000000000000000000000000000")
			d := id("22000000000000000000000000000000")
			q := e.Entities[p]
			q.Fields[fid(0xb103)] = lv(d)
			e.Entities[p] = q
			e.Entities[d] = ent(d, dependencySchema, map[wire.ID]wire.Value{fid(0xb120): bv("missing"), fid(0xb121): bv("v1"), fid(0xb122): rv(id("ffffffffffffffffffffffffffffffff"))})
		},
		"dependency-cycle": func(e *wire.Envelope) {
			p := id("20000000000000000000000000000000")
			d := id("22000000000000000000000000000000")
			q := e.Entities[p]
			q.Fields[fid(0xb103)] = lv(d)
			e.Entities[p] = q
			e.Entities[d] = ent(d, dependencySchema, map[wire.ID]wire.Value{fid(0xb120): bv("self"), fid(0xb121): bv("v1"), fid(0xb122): rv(p)})
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			e := valid()
			mutate(&e)
			if err := ValidateEnvelope(e); err == nil {
				t.Fatal("accepted forgery")
			}
		})
	}
}

func TestRejectsFunctionOwnedByTwoPackages(t *testing.T) {
	e := valid()
	first := id("20000000000000000000000000000000")
	second := id("21000000000000000000000000000000")
	iface := id("31000000000000000000000000000000")
	base := e.Entities[id("30000000000000000000000000000000")]
	base.ID = iface
	e.Entities[iface] = base
	e.Entities[second] = ent(second, packageSchema, map[wire.ID]wire.Value{fid(0xb100): bv("example/other"), fid(0xb101): bv("x"), fid(0xb102): lv(iface), fid(0xb103): lv(), fid(0xb104): lv(), fid(0xb105): lv(), fid(0xb106): lv()})
	r, _ := Revision(e, second)
	q := e.Entities[second]
	q.Fields[fid(0xb101)] = wire.Value{Tag: 5, Bytes: r}
	e.Entities[second] = q
	if err := ValidateEnvelope(e); err == nil || !strings.Contains(err.Error(), "function_owned_twice") {
		t.Fatalf("err=%v first=%s", err, first)
	}
}

func TestDependencyFirstRevisionAndValidation(t *testing.T) {
	e := valid()
	root := id("20000000000000000000000000000000")
	leaf := id("21000000000000000000000000000000")
	dep := id("22000000000000000000000000000000")
	e.Entities[leaf] = ent(leaf, packageSchema, map[wire.ID]wire.Value{fid(0xb100): bv("example/leaf"), fid(0xb101): bv("pending"), fid(0xb102): lv(), fid(0xb103): lv(), fid(0xb104): lv(), fid(0xb105): lv(), fid(0xb106): lv()})
	lr, err := Revision(e, leaf)
	if err != nil {
		t.Fatal(err)
	}
	q := e.Entities[leaf]
	q.Fields[fid(0xb101)] = wire.Value{Tag: 5, Bytes: lr}
	e.Entities[leaf] = q
	e.Entities[dep] = ent(dep, dependencySchema, map[wire.ID]wire.Value{fid(0xb120): bv("example/leaf"), fid(0xb121): bv("exact"), fid(0xb122): rv(leaf)})
	q = e.Entities[root]
	q.Fields[fid(0xb103)] = lv(dep)
	e.Entities[root] = q
	rr, err := Revision(e, root)
	if err != nil {
		t.Fatal(err)
	}
	q = e.Entities[root]
	q.Fields[fid(0xb101)] = wire.Value{Tag: 5, Bytes: rr}
	e.Entities[root] = q
	if err := ValidateEnvelope(e); err != nil {
		t.Fatal(err)
	}
}
