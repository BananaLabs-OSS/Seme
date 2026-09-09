package canonicaleval

import (
	"strings"
	"testing"

	"seme.local/reference/wire"
)

func TestClosureCaptureOwnershipAndBudgetReject(t *testing.T) {
	capture, value, update := id(0x7900), id(0x7901), id(0x7902)
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		value:  {ID: value, Schema: id(0x9070), Fields: map[wire.ID]wire.Value{id(0x9700): {Tag: 3, Unsigned: 1}}},
		update: {ID: update, Schema: id(0xa032), Fields: map[wire.ID]wire.Value{id(0xa0320): {Tag: 6, Reference: capture}, id(0xa0321): {Tag: 6, Reference: value}}},
	}}
	if _, err := eval(g, update, map[wire.ID]Value{}, 8); err == nil || !strings.Contains(err.Error(), "ownership") {
		t.Fatalf("foreign capture accepted: %v", err)
	}
	read := id(0x7903)
	g.Entities[read] = wire.Entity{ID: read, Schema: id(0xa031), Fields: map[wire.ID]wire.Value{id(0xa0310): {Tag: 6, Reference: read}}}
	if _, err := eval(g, read, map[wire.ID]Value{}, 1); err == nil {
		t.Fatal("unbound cyclic capture accepted")
	}
}

func TestClosureCallRejectsKindAndMutabilityForgery(t *testing.T) {
	call, callee := id(0x7910), id(0x7911)
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{call: {ID: call, Schema: id(0xa024), Fields: map[wire.ID]wire.Value{id(0xa0240): {Tag: 6, Reference: callee}, id(0xa0241): {Tag: 7}}}}}
	if _, err := eval(g, call, map[wire.ID]Value{callee: {Kind: "i64", I64: "0"}}, 8); err == nil {
		t.Fatal("non-closure callee accepted")
	}
	g.Entities[callee] = wire.Entity{ID: callee, Schema: id(0x9013), Fields: map[wire.ID]wire.Value{id(0x9130): {Tag: 6, Reference: callee}}}
	mutable := &closureValue{mutable: true, captures: map[wire.ID]Value{}}
	if _, err := eval(g, call, map[wire.ID]Value{callee: {Kind: "closure", closure: mutable}}, 8); err == nil {
		t.Fatal("mutable closure accepted by stateless call")
	}
}

func TestClosureRejectsForgedFunctionAndCaptureTypes(t *testing.T) {
	i64, boolean, functionType, parameter, initial, capture, body, construct := id(0x7920), id(0x7921), id(0x7922), id(0x7923), id(0x7924), id(0x7925), id(0x7926), id(0x7927)
	ref := func(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		i64: {ID: i64, Schema: id(0x9010)}, boolean: {ID: boolean, Schema: id(0x9020)},
		functionType: {ID: functionType, Schema: id(0xa020), Fields: map[wire.ID]wire.Value{id(0xa0200): {Tag: 7, List: []wire.Value{ref(i64)}}, id(0xa0201): ref(i64)}},
		parameter:    {ID: parameter, Schema: id(0x9012), Fields: map[wire.ID]wire.Value{id(0x9121): ref(i64)}},
		initial:      {ID: initial, Schema: id(0x9070), Fields: map[wire.ID]wire.Value{id(0x9700): {Tag: 3, Unsigned: 1}}},
		capture:      {ID: capture, Schema: id(0xa021), Fields: map[wire.ID]wire.Value{id(0xa0211): ref(i64), id(0xa0212): ref(initial)}},
		body:         {ID: body, Schema: id(0xa022), Fields: map[wire.ID]wire.Value{id(0xa0220): ref(capture)}},
		construct:    {ID: construct, Schema: id(0xa023), Fields: map[wire.ID]wire.Value{id(0xa0230): ref(functionType), id(0xa0231): {Tag: 7, List: []wire.Value{ref(parameter)}}, id(0xa0232): {Tag: 7, List: []wire.Value{ref(capture)}}, id(0xa0233): ref(body)}},
	}}
	if _, err := eval(g, construct, map[wire.ID]Value{}, 16); err != nil {
		t.Fatalf("valid closure: %v", err)
	}
	for name, mutate := range map[string]func(*wire.Envelope){
		"parameter": func(g *wire.Envelope) {
			e := g.Entities[parameter]
			e.Fields[id(0x9121)] = ref(boolean)
			g.Entities[parameter] = e
		},
		"capture": func(g *wire.Envelope) {
			e := g.Entities[capture]
			e.Fields[id(0xa0211)] = ref(boolean)
			g.Entities[capture] = e
		},
		"arity": func(g *wire.Envelope) {
			e := g.Entities[functionType]
			e.Fields[id(0xa0200)] = wire.Value{Tag: 7}
			g.Entities[functionType] = e
		},
	} {
		t.Run(name, func(t *testing.T) {
			forged := cloneGraph(g)
			mutate(&forged)
			if _, err := eval(forged, construct, map[wire.ID]Value{}, 16); err == nil {
				t.Fatal("forgery accepted")
			}
		})
	}
	value, _ := eval(g, construct, map[wire.ID]Value{}, 16)
	arguments := wire.Value{Tag: 7, List: []wire.Value{ref(initial)}}
	if _, _, err := invokeClosure(g, value.closure, arguments, map[wire.ID]Value{}, 16); err != nil {
		t.Fatalf("valid invocation: %v", err)
	}
	forged := *value.closure
	forged.resultType = boolean
	if _, _, err := invokeClosure(g, &forged, arguments, map[wire.ID]Value{}, 16); err == nil {
		t.Fatal("forged result type accepted")
	}
}
