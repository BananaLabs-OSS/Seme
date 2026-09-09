package canonicaleval

import (
	"strings"
	"testing"

	"seme.local/reference/wire"
)

func TestFoldTraversesArrayAndSliceDeterministically(t *testing.T) {
	for _, kind := range []string{"array", "slice"} {
		g, root, parameter := foldGraph()
		env := map[wire.ID]Value{parameter: {Kind: kind, Items: []Value{{Kind: "i64", I64: "9223372036854775807"}, {Kind: "i64", I64: "1"}}}}
		got, err := eval(g, root, env, 64)
		if err != nil || got.Kind != "i64" || got.I64 != "-9223372036854775808" {
			t.Fatalf("%s got=%#v err=%v", kind, got, err)
		}
		if _, leaked := env[id(0xf105)]; leaked {
			t.Fatal("iteration binding escaped lexical fold scope")
		}
	}
}

func TestFoldRejectsMalformedStructureTypesCyclesAndBounds(t *testing.T) {
	base, root, parameter := foldGraph()
	valid := map[wire.ID]Value{parameter: {Kind: "slice", Items: []Value{{Kind: "i64", I64: "1"}}}}
	tests := []struct {
		name, want string
		budget     int
		edit       func(*wire.Envelope, map[wire.ID]Value)
	}{
		{"missing collection", "fold_fields", 64, func(g *wire.Envelope, _ map[wire.ID]Value) {
			e := g.Entities[root]
			delete(e.Fields, id(0x9f70))
			g.Entities[root] = e
		}},
		{"wrong collection", "fold_collection", 64, func(_ *wire.Envelope, e map[wire.ID]Value) { e[parameter] = Value{Kind: "bool", Bool: true} }},
		{"wrong initial", "fold_body", 64, func(g *wire.Envelope, _ map[wire.ID]Value) {
			e := g.Entities[id(0xf103)]
			e.Schema = id(0x90b0)
			e.Fields = map[wire.ID]wire.Value{id(0x9b00): {Tag: 1}}
			g.Entities[e.ID] = e
		}},
		{"missing binding", "fold_fields", 64, func(g *wire.Envelope, _ map[wire.ID]Value) { delete(g.Entities, id(0xf105)) }},
		{"wrong binding schema", "fold_fields", 64, func(g *wire.Envelope, _ map[wire.ID]Value) {
			e := g.Entities[id(0xf105)]
			e.Schema = id(0x9020)
			g.Entities[e.ID] = e
		}},
		{"body cycle", "budget", 64, func(g *wire.Envelope, _ map[wire.ID]Value) {
			e := g.Entities[root]
			e.Fields[id(0x9f74)] = wire.Value{Tag: 6, Reference: root}
			g.Entities[root] = e
		}},
		{"budget", "budget", 1, func(_ *wire.Envelope, _ map[wire.ID]Value) {}},
		{"over bound", "fold_collection", 64, func(_ *wire.Envelope, e map[wire.ID]Value) {
			items := make([]Value, 513)
			for i := range items {
				items[i] = Value{Kind: "i64", I64: "0"}
			}
			e[parameter] = Value{Kind: "slice", Items: items}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := cloneGraph(base)
			env := cloneEnv(valid)
			tc.edit(&g, env)
			_, err := eval(g, root, env, tc.budget)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want=%s", err, tc.want)
			}
		})
	}
}

func foldGraph() (wire.Envelope, wire.ID, wire.ID) {
	i64, parameter, read, initial, accumulator, element, left, right, body, fold := id(0xf100), id(0xf101), id(0xf102), id(0xf103), id(0xf104), id(0xf105), id(0xf106), id(0xf107), id(0xf108), id(0xf109)
	entities := map[wire.ID]wire.Entity{
		i64: {ID: i64, Schema: id(0x9010)}, parameter: {ID: parameter, Schema: id(0x9012)}, read: {ID: read, Schema: id(0x9013), Fields: map[wire.ID]wire.Value{id(0x9130): {Tag: 6, Reference: parameter}}},
		initial:     {ID: initial, Schema: id(0x9070), Fields: map[wire.ID]wire.Value{id(0x9700): {Tag: 3}, id(0x9701): {Tag: 6, Reference: i64}}},
		accumulator: {ID: accumulator, Schema: id(0x90f5)}, element: {ID: element, Schema: id(0x90f5)}, left: {ID: left, Schema: id(0x90f6), Fields: map[wire.ID]wire.Value{id(0x9f60): {Tag: 6, Reference: accumulator}}}, right: {ID: right, Schema: id(0x90f6), Fields: map[wire.ID]wire.Value{id(0x9f60): {Tag: 6, Reference: element}}},
		body: {ID: body, Schema: id(0x9014), Fields: map[wire.ID]wire.Value{id(0x9140): {Tag: 6, Reference: left}, id(0x9141): {Tag: 6, Reference: right}, id(0x9142): {Tag: 6, Reference: i64}}}, fold: {ID: fold, Schema: id(0x90f7), Fields: map[wire.ID]wire.Value{id(0x9f70): {Tag: 6, Reference: read}, id(0x9f71): {Tag: 6, Reference: initial}, id(0x9f72): {Tag: 6, Reference: accumulator}, id(0x9f73): {Tag: 6, Reference: element}, id(0x9f74): {Tag: 6, Reference: body}}},
	}
	return wire.Envelope{Entities: entities}, fold, parameter
}
