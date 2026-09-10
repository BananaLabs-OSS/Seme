package canonicaleval

import (
	"testing"

	"seme.local/reference/wire"
)

func TestExpressionTypeDerivesUniqueStringLiteralType(t *testing.T) {
	typ, literal := id(0x7101), id(0x7102)
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		typ:     {ID: typ, Schema: id(0x9040), Version: 1, Fields: map[wire.ID]wire.Value{}},
		literal: {ID: literal, Schema: id(0x9050), Version: 1, Fields: map[wire.ID]wire.Value{id(0x9500): {Tag: 5, Bytes: []byte("value")}}},
	}}
	if got, ok := ExpressionType(g, literal); !ok || got != typ {
		t.Fatalf("string literal type = %s,%v", got, ok)
	}
	g.Entities[id(0x7103)] = wire.Entity{ID: id(0x7103), Schema: id(0x9040), Version: 1, Fields: map[wire.ID]wire.Value{}}
	if _, ok := ExpressionType(g, literal); ok {
		t.Fatal("ambiguous string type accepted")
	}
}
