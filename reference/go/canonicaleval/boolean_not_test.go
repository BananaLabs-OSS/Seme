package canonicaleval

import (
	"strings"
	"testing"

	"seme.local/reference/wire"
)

func TestBooleanNotTruthTableAndType(t *testing.T) {
	for _, value := range []bool{false, true} {
		g, root := booleanNotGraph(value)
		got, err := EvaluateExpression(g, root)
		if err != nil {
			t.Fatal(err)
		}
		if got.Kind != "bool" || got.Bool == value {
			t.Fatalf("not(%t)=%#v", value, got)
		}
		typeID, ok := ExpressionType(g, root)
		if !ok || typeID != id(0xe001) {
			t.Fatalf("type=%x ok=%t", typeID, ok)
		}
	}
}

func TestBooleanNotRejectsMalformedOperands(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(wire.Envelope, wire.ID)
	}{
		{"missing", func(g wire.Envelope, root wire.ID) {
			e := g.Entities[root]
			delete(e.Fields, id(0xa0690))
			g.Entities[root] = e
		}},
		{"wrong-tag", func(g wire.Envelope, root wire.ID) {
			e := g.Entities[root]
			e.Fields[id(0xa0690)] = wire.Value{Tag: 3, Unsigned: 1}
			g.Entities[root] = e
		}},
		{"extra-field", func(g wire.Envelope, root wire.ID) {
			e := g.Entities[root]
			e.Fields[id(0xffff)] = wire.Value{Tag: 1}
			g.Entities[root] = e
		}},
		{"non-boolean", func(g wire.Envelope, root wire.ID) {
			i64 := id(0xe004)
			literal := id(0xe005)
			g.Entities[i64] = wire.Entity{ID: i64, Schema: id(0x9010), Fields: map[wire.ID]wire.Value{}}
			g.Entities[literal] = wire.Entity{ID: literal, Schema: id(0x9070), Fields: map[wire.ID]wire.Value{id(0x9700): {Tag: 3}, id(0x9701): {Tag: 6, Reference: i64}}}
			e := g.Entities[root]
			e.Fields[id(0xa0690)] = wire.Value{Tag: 6, Reference: literal}
			g.Entities[root] = e
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			g, root := booleanNotGraph(true)
			test.edit(g, root)
			_, err := EvaluateExpression(g, root)
			if err == nil || !strings.Contains(err.Error(), "boolean_not") {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestBooleanNotTypeRequiresUniqueBooleanType(t *testing.T) {
	g, root := booleanNotGraph(true)
	duplicate := id(0xe006)
	g.Entities[duplicate] = wire.Entity{ID: duplicate, Schema: id(0x9020), Fields: map[wire.ID]wire.Value{}}
	if _, ok := ExpressionType(g, root); ok {
		t.Fatal("ambiguous boolean type accepted")
	}
}

func booleanNotGraph(value bool) (wire.Envelope, wire.ID) {
	boolean, literal, root := id(0xe001), id(0xe002), id(0xe003)
	truth := byte(1)
	if value {
		truth = 2
	}
	return wire.Envelope{Entities: map[wire.ID]wire.Entity{
		boolean: {ID: boolean, Schema: id(0x9020), Fields: map[wire.ID]wire.Value{}},
		literal: {ID: literal, Schema: id(0x90b0), Fields: map[wire.ID]wire.Value{id(0x9b00): {Tag: truth}}},
		root:    {ID: root, Schema: id(0xa069), Fields: map[wire.ID]wire.Value{id(0xa0690): {Tag: 6, Reference: literal}}},
	}}, root
}
