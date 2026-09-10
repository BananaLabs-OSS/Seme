package configurationexecutor

import (
	"testing"

	"seme.local/reference/canonicaleval"
	"seme.local/reference/wire"
)

func TestResolveEveryBoundSourceArmAndRecord(t *testing.T) {
	i64, fieldA, fieldB, fieldC, fieldD, record := id("1"), id("2"), id("3"), id("4"), id("5"), id("6")
	configField, runtime, predecessor := id("10"), id("11"), id("12")
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	g.Entities[i64] = entity(i64, id("9010"), nil)
	for index, f := range []wire.ID{fieldA, fieldB, fieldC, fieldD} {
		g.Entities[f] = entity(f, id("9031"), map[wire.ID]wire.Value{id("9310"): blob([]byte{'a' + byte(index)}), id("9311"): ref(i64), id("9312"): uintv(uint64(index))})
	}
	g.Entities[record] = entity(record, id("9030"), map[wire.ID]wire.Value{id("9300"): blob([]byte("Composite")), id("9301"): list(fieldA, fieldB, fieldC, fieldD)})
	literal := id("20")
	g.Entities[literal] = entity(literal, id("9070"), map[wire.ID]wire.Value{id("9700"): uintv(4), id("9701"): ref(i64)})
	configSource := source(g.Entities, id("30"), 0, i64, id("41a2"), configField)
	runtimeSource := source(g.Entities, id("31"), 1, i64, id("41a3"), runtime)
	predSource := source(g.Entities, id("32"), 2, i64, id("41a4"), predecessor)
	staticSource := source(g.Entities, id("33"), 3, i64, id("41a5"), literal)
	assembly := id("40")
	members := []wire.ID{}
	for index, pair := range []struct{ field, source wire.ID }{{fieldA, configSource}, {fieldB, runtimeSource}, {fieldC, predSource}, {fieldD, staticSource}} {
		x := id(string([]byte{'5', byte('0' + index)}))
		g.Entities[x] = entity(x, id("401d"), map[wire.ID]wire.Value{id("41d0"): ref(pair.field), id("41d1"): ref(pair.source)})
		members = append(members, x)
	}
	g.Entities[assembly] = entity(assembly, id("401c"), map[wire.ID]wire.Value{id("41c0"): ref(record), id("41c1"): list(members...)})
	recordSource := source(g.Entities, id("34"), 4, record, id("41a6"), assembly)
	p := plan{graph: g, fields: map[wire.ID]canonicaleval.Value{configField: i64v("1")}, runtime: map[wire.ID]canonicaleval.Value{runtime: i64v("2")}, static: map[wire.ID]canonicaleval.Value{}}
	if err := preflightSource(&p, recordSource, map[wire.ID]bool{}); err != nil {
		t.Fatal(err)
	}
	got, err := resolveSource(p, recordSource, map[wire.ID]canonicaleval.Value{predecessor: i64v("3")}, map[wire.ID]bool{})
	if err != nil || got.Kind != "record" || got.Fields["a"].I64 != "1" || got.Fields["b"].I64 != "2" || got.Fields["c"].I64 != "3" || got.Fields["d"].I64 != "4" {
		t.Fatalf("got=%#v err=%v", got, err)
	}
	if _, err = resolveSource(p, predSource, nil, map[wire.ID]bool{}); err == nil {
		t.Fatal("missing predecessor accepted")
	}
	bad := p
	bad.static = map[wire.ID]canonicaleval.Value{}
	q := bad.graph.Entities[literal]
	q.Fields[id("9701")] = ref(record)
	bad.graph.Entities[literal] = q
	if err = preflightSource(&bad, staticSource, map[wire.ID]bool{}); err == nil {
		t.Fatal("forged static type accepted")
	}
}

func TestExecuteRejectsUnauthenticatedWithoutOutputs(t *testing.T) {
	got, err := Execute(Input{RuntimeInputs: map[string]canonicaleval.Value{"extra": i64v("1")}})
	if err == nil || len(got.Outputs) != 0 || len(got.Lifecycle) != 0 {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}

func TestExecuteV3RejectsUnauthenticatedWithoutOutputs(t *testing.T) {
	got, err := ExecuteV3(V3Input{RuntimeInputs: map[string]canonicaleval.Value{"application-state": i64v("1")}})
	if err == nil || len(got.Outputs) != 0 || len(got.Lifecycle) != 0 {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}

func source(es map[wire.ID]wire.Entity, x wire.ID, kind uint64, typ, arm, target wire.ID) wire.ID {
	k := id("8" + x.String()[31:])
	es[k] = entity(k, id("4019"), map[wire.ID]wire.Value{id("4190"): uintv(kind)})
	es[x] = entity(x, id("401a"), map[wire.ID]wire.Value{id("41a0"): ref(k), id("41a1"): ref(typ), arm: ref(target)})
	return x
}
func entity(x, s wire.ID, fields map[wire.ID]wire.Value) wire.Entity {
	return wire.Entity{ID: x, Schema: s, Version: 1, Fields: fields}
}
func ref(x wire.ID) wire.Value  { return wire.Value{Tag: 6, Reference: x} }
func blob(x []byte) wire.Value  { return wire.Value{Tag: 5, Bytes: x} }
func uintv(x uint64) wire.Value { return wire.Value{Tag: 3, Unsigned: x} }
func list(xs ...wire.ID) wire.Value {
	v := wire.Value{Tag: 7}
	for _, x := range xs {
		v.List = append(v.List, ref(x))
	}
	return v
}
func i64v(x string) canonicaleval.Value { return canonicaleval.Value{Kind: "i64", I64: x} }
