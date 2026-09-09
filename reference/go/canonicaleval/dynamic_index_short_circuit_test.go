package canonicaleval

import (
	"strings"
	"testing"

	"seme.local/reference/wire"
)

// TestDynamicIndexShortCircuitSemantics deliberately puts a failing dynamic
// read on the right of both Boolean operators.  A truth-table-only test would
// not distinguish correct left-to-right evaluation from eager evaluation.
func TestDynamicIndexShortCircuitSemantics(t *testing.T) {
	for _, tc := range []struct {
		name   string
		schema uint64
		left   bool
		want   bool
		fails  bool
	}{
		{"false and skips out of bounds", 0x90b1, false, false, false},
		{"true or skips out of bounds", 0x90c1, true, true, false},
		{"true and evaluates out of bounds", 0x90b1, true, false, true},
		{"false or evaluates out of bounds", 0x90c1, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, root, env := dynamicIndexBooleanGraph(tc.schema, tc.left, "2")
			got, err := eval(g, root, env, 64)
			if tc.fails {
				if err == nil {
					t.Fatalf("error=%v", err)
				}
				return
			}
			if err != nil || got.Kind != "bool" || got.Bool != tc.want {
				t.Fatalf("got=%#v error=%v", got, err)
			}
		})
	}
}

func TestDynamicIndexInRangeAndMalformedGraphs(t *testing.T) {
	g, _, env := dynamicIndexBooleanGraph(0x90b1, true, "1")
	got, err := eval(g, id(0xd107), env, 32)
	if err != nil || got.Kind != "i64" || got.I64 != "20" {
		t.Fatalf("in-range=%#v error=%v", got, err)
	}

	mutations := []struct {
		name string
		edit func(*wire.Envelope)
		want string
	}{
		{"missing index field", func(g *wire.Envelope) {
			e := g.Entities[id(0xd107)]
			delete(e.Fields, id(0x9fa1))
			g.Entities[e.ID] = e
		}, "dynamic_index_fields"},
		{"negative bound", func(g *wire.Envelope) {
			e := g.Entities[id(0xd106)]
			e.Fields[id(0x9700)] = wire.Value{Tag: 3, Unsigned: ^uint64(0)}
			g.Entities[e.ID] = e
		}, "dynamic_index"},
		{"wrong collection type", func(g *wire.Envelope) {
			e := g.Entities[id(0xd107)]
			e.Fields[id(0x9fa0)] = wire.Value{Tag: 6, Reference: id(0xd106)}
			g.Entities[e.ID] = e
		}, "dynamic_index"},
		{"expression cycle", func(g *wire.Envelope) {
			e := g.Entities[id(0xd107)]
			e.Fields[id(0x9fa1)] = wire.Value{Tag: 6, Reference: e.ID}
			g.Entities[e.ID] = e
		}, "dynamic_index"},
		{"foreign parameter ownership", func(g *wire.Envelope) {
			e := g.Entities[id(0xd104)]
			e.Fields[id(0x9130)] = wire.Value{Tag: 6, Reference: id(0xd1ff)}
			g.Entities[e.ID] = e
		}, "dynamic_index"},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			copy := cloneGraph(g)
			tc.edit(&copy)
			_, err := eval(copy, id(0xd107), env, 16)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want=%s", err, tc.want)
			}
		})
	}
}

func dynamicIndexBooleanGraph(booleanSchema uint64, left bool, index string) (wire.Envelope, wire.ID, map[wire.ID]Value) {
	i64, sliceType := id(0xd100), id(0xd101)
	collectionParam, leftLiteral := id(0xd102), id(0xd103)
	collectionRead, indexType, indexLiteral := id(0xd104), id(0xd105), id(0xd106)
	dynamicRead, zero, comparison, root := id(0xd107), id(0xd108), id(0xd109), id(0xd10a)
	leftTag := byte(1)
	if left {
		leftTag = 2
	}
	indexValue, _ := parseI64(index)
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		i64:             {ID: i64, Schema: id(0x9010)},
		sliceType:       {ID: sliceType, Schema: id(0x90f8), Fields: map[wire.ID]wire.Value{id(0x9f80): {Tag: 6, Reference: i64}}},
		collectionParam: {ID: collectionParam, Schema: id(0x9012), Fields: map[wire.ID]wire.Value{id(0x9121): {Tag: 6, Reference: sliceType}}},
		leftLiteral:     {ID: leftLiteral, Schema: id(0x90b0), Fields: map[wire.ID]wire.Value{id(0x9b00): {Tag: leftTag}}},
		collectionRead:  {ID: collectionRead, Schema: id(0x9013), Fields: map[wire.ID]wire.Value{id(0x9130): {Tag: 6, Reference: collectionParam}}},
		indexType:       {ID: indexType, Schema: id(0x9010)},
		indexLiteral:    {ID: indexLiteral, Schema: id(0x9070), Fields: map[wire.ID]wire.Value{id(0x9700): indexValue, id(0x9701): {Tag: 6, Reference: indexType}}},
		dynamicRead:     {ID: dynamicRead, Schema: id(0x90fa), Fields: map[wire.ID]wire.Value{id(0x9fa0): {Tag: 6, Reference: collectionRead}, id(0x9fa1): {Tag: 6, Reference: indexLiteral}}},
		zero:            {ID: zero, Schema: id(0x9070), Fields: map[wire.ID]wire.Value{id(0x9700): {Tag: 3}, id(0x9701): {Tag: 6, Reference: i64}}},
		comparison:      {ID: comparison, Schema: id(0x9021), Fields: map[wire.ID]wire.Value{id(0x9160): {Tag: 6, Reference: dynamicRead}, id(0x9161): {Tag: 6, Reference: zero}}},
		root:            {ID: root, Schema: id(booleanSchema), Fields: map[wire.ID]wire.Value{id(map[uint64]uint64{0x90b1: 0x9b10, 0x90c1: 0x9c10}[booleanSchema]): {Tag: 6, Reference: leftLiteral}, id(map[uint64]uint64{0x90b1: 0x9b11, 0x90c1: 0x9c11}[booleanSchema]): {Tag: 6, Reference: comparison}}},
	}}
	return g, root, map[wire.ID]Value{collectionParam: {Kind: "slice", Items: []Value{{Kind: "i64", I64: "10"}, {Kind: "i64", I64: "20"}}}}
}

func parseI64(value string) (wire.Value, bool) {
	if value == "-1" {
		return wire.Value{Tag: 3, Unsigned: ^uint64(0)}, true
	}
	var n uint64
	for _, r := range value {
		n = n*10 + uint64(r-'0')
	}
	return wire.Value{Tag: 3, Unsigned: n}, true
}
