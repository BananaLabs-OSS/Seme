package wasmtarget

import (
	"testing"

	"seme.local/reference/wire"
)

func TestComposedResultConstructorRejectsWrongArmAndPayloadTypes(t *testing.T) {
	i64, boolean, resultType, value, constructor := identity(0x7a00), identity(0x7a01), identity(0x7a02), identity(0x7a03), identity(0x7a04)
	ref := func(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
	base := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		i64:         {ID: i64, Schema: identity(0x9010), Fields: map[wire.ID]wire.Value{identity(0x9100): {Tag: 3, Unsigned: 64}, identity(0x9101): {Tag: 2}, identity(0x9102): {Tag: 3}}},
		boolean:     {ID: boolean, Schema: identity(0x9020)},
		resultType:  {ID: resultType, Schema: identity(0x9042), Fields: map[wire.ID]wire.Value{identity(0x9400): ref(i64), identity(0x9401): ref(i64)}},
		value:       {ID: value, Schema: identity(0x9070), Fields: map[wire.ID]wire.Value{identity(0x9700): {Tag: 3, Unsigned: 7}}},
		constructor: {ID: constructor, Schema: identity(0x9043), Fields: map[wire.ID]wire.Value{identity(0x9410): ref(resultType), identity(0x9411): ref(value)}},
	}}
	lower := func(g wire.Envelope) error {
		_, err := (&composedLowerer{graph: g, budget: 32}).expr(constructor, composedScope{})
		return err
	}
	if err := lower(base); err != nil {
		t.Fatalf("valid constructor: %v", err)
	}
	for name, mutate := range map[string]func(*wire.Envelope){
		"non-i64 error arm": func(g *wire.Envelope) {
			e := g.Entities[resultType]
			e.Fields[identity(0x9401)] = ref(boolean)
			g.Entities[resultType] = e
		},
		"wrong payload kind": func(g *wire.Envelope) {
			e := g.Entities[value]
			e.Schema = identity(0x9050)
			e.Fields = map[wire.ID]wire.Value{identity(0x9500): {Tag: 2}}
			g.Entities[value] = e
		},
		"foreign result type": func(g *wire.Envelope) {
			e := g.Entities[constructor]
			e.Fields[identity(0x9410)] = ref(boolean)
			g.Entities[constructor] = e
		},
	} {
		t.Run(name, func(t *testing.T) {
			g := cloneWireGraph(base)
			mutate(&g)
			if err := lower(g); err == nil {
				t.Fatal("forgery accepted")
			}
		})
	}
}
