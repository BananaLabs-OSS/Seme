package canonicaleval

import (
	"fmt"
	"strings"
	"testing"

	"seme.local/reference/wire"
)

func TestImmutableCollectionOperations(t *testing.T) {
	g, env, x := collectionOperationGraph()
	tests := []struct {
		name       string
		expression wire.ID
		want       []string
	}{
		{"append", x["append"], []string{"1", "2", "3"}}, {"update", x["update"], []string{"1", "9"}}, {"remove", x["remove"], []string{"2"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := eval(g, test.expression, cloneEnv(env), 128)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Items) != len(test.want) {
				t.Fatalf("%#v", got)
			}
			for i, w := range test.want {
				if got.Items[i].I64 != w {
					t.Fatalf("%#v", got)
				}
			}
			got.Items[0].I64 = "77"
			if env[x["sliceParam"]].Items[0].I64 != "1" {
				t.Fatal("aliased input")
			}
		})
	}
	updated, err := eval(g, x["mapUpdate"], cloneEnv(env), 128)
	if err != nil || len(updated.Entries) != 2 || updated.Entries[0].Value.I64 != "9" {
		t.Fatalf("update=%#v %v", updated, err)
	}
	removed, err := eval(g, x["mapRemove"], cloneEnv(env), 128)
	if err != nil || len(removed.Entries) != 1 {
		t.Fatalf("remove=%#v %v", removed, err)
	}
	missing, err := eval(g, x["mapMissing"], cloneEnv(env), 128)
	if err != nil || len(missing.Entries) != 2 {
		t.Fatalf("missing=%#v %v", missing, err)
	}
	missing.Entries[0].Value.I64 = "88"
	if env[x["mapParam"]].Entries[0].Value.I64 != "4" {
		t.Fatal("map aliased input")
	}
}

func TestCollectionOperationBoundsAndMalformedGraphs(t *testing.T) {
	g, env, x := collectionOperationGraph()
	for _, name := range []string{"negativeRemove", "largeUpdate"} {
		if _, err := eval(g, x[name], cloneEnv(env), 64); err == nil || !strings.Contains(err.Error(), "index") {
			t.Errorf("%s: %v", name, err)
		}
	}
	full := make([]Value, 512)
	for i := range full {
		full[i] = Value{Kind: "i64", I64: "0"}
	}
	fullEnv := cloneEnv(env)
	fullEnv[x["sliceParam"]] = Value{Kind: "slice", Items: full}
	if _, err := eval(g, x["append"], fullEnv, 128); err == nil || !strings.Contains(err.Error(), "append") {
		t.Fatalf("full append: %v", err)
	}
	fullMap := make([]Entry, 512)
	for i := range fullMap {
		fullMap[i] = Entry{Key: Value{Kind: "i64", I64: fmt.Sprint(i + 10)}, Value: Value{Kind: "i64", I64: "0"}}
	}
	mapEnv := cloneEnv(env)
	mapEnv[x["mapParam"]] = Value{Kind: "map", ValueType: "i64", Entries: fullMap}
	if _, err := eval(g, x["mapUpdate"], mapEnv, 128); err == nil || !strings.Contains(err.Error(), "size") {
		t.Fatalf("full map update: %v", err)
	}
	wrongElement := cloneEnv(env)
	wrongElement[x["sliceParam"]] = Value{Kind: "slice", Items: []Value{{Kind: "text", Text: "x"}}}
	if _, err := eval(g, x["append"], wrongElement, 128); err == nil || !strings.Contains(err.Error(), "type") {
		t.Fatalf("wrong element type: %v", err)
	}
	wrongKey := cloneEnv(env)
	wrongKey[x["mapParam"]] = Value{Kind: "map", ValueType: "i64", Entries: []Entry{{Key: Value{Kind: "text", Text: "x"}, Value: Value{Kind: "i64", I64: "1"}}}}
	if _, err := eval(g, x["mapMissing"], wrongKey, 128); err == nil || !strings.Contains(err.Error(), "type") {
		t.Fatalf("wrong key type: %v", err)
	}
	wrongValue := cloneEnv(env)
	wrongValue[x["mapParam"]] = Value{Kind: "map", ValueType: "i64", Entries: []Entry{{Key: Value{Kind: "i64", I64: "1"}, Value: Value{Kind: "text", Text: "x"}}}}
	if _, err := eval(g, x["mapMissing"], wrongValue, 128); err == nil || !strings.Contains(err.Error(), "type") {
		t.Fatalf("wrong value type: %v", err)
	}
	mutations := []struct {
		name string
		edit func(*wire.Envelope)
	}{{"missing-field", func(h *wire.Envelope) {
		e := h.Entities[x["append"]]
		delete(e.Fields, id(0x9fb1))
		h.Entities[e.ID] = e
	}}, {"wrong-tag", func(h *wire.Envelope) {
		e := h.Entities[x["remove"]]
		e.Fields[id(0xa0661)] = wire.Value{Tag: 1}
		h.Entities[e.ID] = e
	}}, {"missing-ref", func(h *wire.Envelope) {
		e := h.Entities[x["mapUpdate"]]
		e.Fields[id(0xa0432)] = wire.Value{Tag: 6, Reference: id(0xffff)}
		h.Entities[e.ID] = e
	}}}
	for _, m := range mutations {
		t.Run(m.name, func(t *testing.T) {
			h := cloneEnvelope(g)
			m.edit(&h)
			if _, err := eval(h, x[map[string]string{"missing-field": "append", "wrong-tag": "remove", "missing-ref": "mapUpdate"}[m.name]], cloneEnv(env), 64); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	if got, err := eval(g, x["append"], cloneEnv(env), 64); err != nil || len(got.Items) != 3 {
		t.Fatalf("baseline corrupted after mutations: %#v %v", got, err)
	}
	if _, err := eval(g, x["append"], cloneEnv(env), 1); err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("budget: %v", err)
	}
	cycle := cloneEnvelope(g)
	cyclic := cycle.Entities[x["append"]]
	cyclic.Fields[id(0x9fb0)] = wire.Value{Tag: 6, Reference: x["append"]}
	cycle.Entities[cyclic.ID] = cyclic
	if _, err := eval(cycle, x["append"], cloneEnv(env), 16); err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("cycle did not exhaust recursion budget: %v", err)
	}
}

func cloneEnvelope(source wire.Envelope) wire.Envelope {
	result := source
	result.Entities = make(map[wire.ID]wire.Entity, len(source.Entities))
	for key, entity := range source.Entities {
		copyEntity := entity
		copyEntity.Fields = make(map[wire.ID]wire.Value, len(entity.Fields))
		for fieldID, value := range entity.Fields {
			copyEntity.Fields[fieldID] = cloneWireValue(value)
		}
		result.Entities[key] = copyEntity
	}
	return result
}

func cloneWireValue(value wire.Value) wire.Value {
	result := value
	result.Bytes = append([]byte(nil), value.Bytes...)
	result.List = make([]wire.Value, len(value.List))
	for index, item := range value.List {
		result.List[index] = cloneWireValue(item)
	}
	if value.Record != nil {
		result.Record = make(map[wire.ID]wire.Value, len(value.Record))
		for key, item := range value.Record {
			result.Record[key] = cloneWireValue(item)
		}
	}
	return result
}

func collectionOperationGraph() (wire.Envelope, map[wire.ID]Value, map[string]wire.ID) {
	i64, sliceT, mapT := id(0xb001), id(0xb002), id(0xb003)
	sp, mp := id(0xb004), id(0xb005)
	sr, mr := id(0xb006), id(0xb007)
	lit := func(n uint64) wire.Entity {
		z := id(0xb100 + n)
		return wire.Entity{ID: z, Schema: id(0x9070), Fields: map[wire.ID]wire.Value{id(0x9700): {Tag: 3, Unsigned: n}, id(0x9701): {Tag: 6, Reference: i64}}}
	}
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{i64: {ID: i64, Schema: id(0x9010), Fields: map[wire.ID]wire.Value{id(0x9100): {Tag: 3, Unsigned: 64}, id(0x9101): {Tag: 2}, id(0x9102): {Tag: 3}}}, sliceT: {ID: sliceT, Schema: id(0x90f8), Fields: map[wire.ID]wire.Value{id(0x9f80): {Tag: 6, Reference: i64}}}, mapT: {ID: mapT, Schema: id(0xa040), Fields: map[wire.ID]wire.Value{id(0xa0400): {Tag: 6, Reference: i64}, id(0xa0401): {Tag: 6, Reference: i64}}}, sp: {ID: sp, Schema: id(0x9012), Fields: map[wire.ID]wire.Value{id(0x9121): {Tag: 6, Reference: sliceT}}}, mp: {ID: mp, Schema: id(0x9012), Fields: map[wire.ID]wire.Value{id(0x9121): {Tag: 6, Reference: mapT}}}, sr: {ID: sr, Schema: id(0x9013), Fields: map[wire.ID]wire.Value{id(0x9130): {Tag: 6, Reference: sp}}}, mr: {ID: mr, Schema: id(0x9013), Fields: map[wire.ID]wire.Value{id(0x9130): {Tag: 6, Reference: mp}}}}}
	for _, n := range []uint64{0, 1, 2, 3, 9} {
		e := lit(n)
		g.Entities[e.ID] = e
	}
	neg := id(0xb1ff)
	g.Entities[neg] = wire.Entity{ID: neg, Schema: id(0x9070), Fields: map[wire.ID]wire.Value{id(0x9700): {Tag: 3, Unsigned: ^uint64(0)}, id(0x9701): {Tag: 6, Reference: i64}}}
	x := map[string]wire.ID{"sliceParam": sp, "mapParam": mp}
	add := func(name string, schema uint64, fields map[uint64]wire.ID) {
		eid := id(0xb200 + uint64(len(x)))
		f := map[wire.ID]wire.Value{}
		for k, v := range fields {
			f[id(k)] = wire.Value{Tag: 6, Reference: v}
		}
		g.Entities[eid] = wire.Entity{ID: eid, Schema: id(schema), Fields: f}
		x[name] = eid
	}
	add("append", 0x90fb, map[uint64]wire.ID{0x9fb0: sr, 0x9fb1: id(0xb103)})
	add("update", 0x90fc, map[uint64]wire.ID{0x9fc0: sr, 0x9fc1: id(0xb101), 0x9fc2: id(0xb109)})
	add("remove", 0xa066, map[uint64]wire.ID{0xa0660: sr, 0xa0661: id(0xb100)})
	add("negativeRemove", 0xa066, map[uint64]wire.ID{0xa0660: sr, 0xa0661: neg})
	add("largeUpdate", 0x90fc, map[uint64]wire.ID{0x9fc0: sr, 0x9fc1: id(0xb109), 0x9fc2: id(0xb100)})
	add("mapUpdate", 0xa043, map[uint64]wire.ID{0xa0430: mr, 0xa0431: id(0xb101), 0xa0432: id(0xb109)})
	add("mapRemove", 0xa067, map[uint64]wire.ID{0xa0670: mr, 0xa0671: id(0xb102)})
	add("mapMissing", 0xa067, map[uint64]wire.ID{0xa0670: mr, 0xa0671: id(0xb103)})
	env := map[wire.ID]Value{sp: {Kind: "slice", Items: []Value{{Kind: "i64", I64: "1"}, {Kind: "i64", I64: "2"}}}, mp: {Kind: "map", ValueType: "i64", Entries: []Entry{{Key: Value{Kind: "i64", I64: "1"}, Value: Value{Kind: "i64", I64: "4"}}, {Key: Value{Kind: "i64", I64: "2"}, Value: Value{Kind: "i64", I64: "5"}}}}}
	return g, env, x
}
