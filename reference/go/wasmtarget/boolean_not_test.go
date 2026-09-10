package wasmtarget

import (
	"bytes"
	"strings"
	"testing"

	"seme.local/reference/wire"
)

func TestBooleanNotLowersToI32Eqz(t *testing.T) {
	literal, root := identity(0xd001), identity(0xd002)
	graph := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		literal: {ID: literal, Schema: identity(0x90b0), Fields: map[wire.ID]wire.Value{identity(0x9b00): {Tag: 2}}},
		root:    {ID: root, Schema: identity(0xa069), Fields: map[wire.ID]wire.Value{identity(0xa0690): ref(literal)}},
	}}
	budget := 8
	if err := validatePureExpression(graph, root, "bool", nil, map[wire.ID]bool{}, &budget); err != nil {
		t.Fatal(err)
	}
	budget = 8
	code, err := lowerHelperBoolean(graph, root, nil, map[byte]bool{}, map[wire.ID]bool{}, &budget)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(code, []byte{0x41, 0x01, 0x45}) {
		t.Fatalf("BooleanNot instructions=%x", code)
	}
}

func TestBooleanNotRejectsMalformedAndNonBooleanOperands(t *testing.T) {
	literal, root := identity(0xd011), identity(0xd012)
	base := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		literal: {ID: literal, Schema: identity(0x9070), Fields: map[wire.ID]wire.Value{}},
		root:    {ID: root, Schema: identity(0xa069), Fields: map[wire.ID]wire.Value{identity(0xa0690): ref(literal)}},
	}}
	for _, test := range []struct {
		name    string
		edit    func(*wire.Entity)
		problem string
	}{
		{"non-boolean", func(*wire.Entity) {}, "integer_literal_type"},
		{"missing", func(entity *wire.Entity) { delete(entity.Fields, identity(0xa0690)) }, "boolean_not_fields"},
		{"wrong-tag", func(entity *wire.Entity) { entity.Fields[identity(0xa0690)] = wire.Value{Tag: 3, Unsigned: 1} }, "boolean_not_fields"},
	} {
		t.Run(test.name, func(t *testing.T) {
			graph := cloneTargetGraph(base)
			entity := graph.Entities[root]
			test.edit(&entity)
			graph.Entities[root] = entity
			budget := 8
			err := validatePureExpression(graph, root, "bool", nil, map[wire.ID]bool{}, &budget)
			if err == nil || !strings.Contains(err.Error(), test.problem) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
