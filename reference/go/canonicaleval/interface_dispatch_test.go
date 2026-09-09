package canonicaleval

import (
	"strings"
	"testing"

	"seme.local/reference/wire"
)

func TestMethodRequirementRejectsEveryForgedRelationship(t *testing.T) {
	i64, concrete, other := id(0x8100), id(0x8101), id(0x8102)
	receiver, parameter, method, requirement := id(0x8110), id(0x8111), id(0x8112), id(0x8113)
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		i64: {ID: i64, Schema: id(0x9010)}, concrete: {ID: concrete, Schema: id(0x9030)}, other: {ID: other, Schema: id(0x9030)},
		receiver:    {ID: receiver, Schema: id(0xa000), Fields: map[wire.ID]wire.Value{id(0xa0001): {Tag: 6, Reference: concrete}}},
		parameter:   {ID: parameter, Schema: id(0x9012), Fields: map[wire.ID]wire.Value{id(0x9121): {Tag: 6, Reference: i64}}},
		method:      {ID: method, Schema: id(0xa002), Fields: map[wire.ID]wire.Value{id(0xa0020): {Tag: 5, Bytes: []byte("Adjust")}, id(0xa0021): {Tag: 6, Reference: receiver}, id(0xa0022): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: parameter}}}, id(0xa0023): {Tag: 6, Reference: i64}}},
		requirement: {ID: requirement, Schema: id(0xa011), Fields: map[wire.ID]wire.Value{id(0xa0110): {Tag: 5, Bytes: []byte("Adjust")}, id(0xa0111): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: i64}}}, id(0xa0112): {Tag: 6, Reference: i64}}},
	}}
	mutations := map[string]func(*wire.Envelope){
		"method membership": func(h *wire.Envelope) { e := h.Entities[method]; e.Schema = id(0x9011); h.Entities[method] = e },
		"requirement membership": func(h *wire.Envelope) {
			e := h.Entities[requirement]
			e.Schema = id(0x9011)
			h.Entities[requirement] = e
		},
		"name": func(h *wire.Envelope) {
			e := h.Entities[requirement]
			e.Fields[id(0xa0110)] = wire.Value{Tag: 5, Bytes: []byte("Other")}
			h.Entities[requirement] = e
		},
		"arity": func(h *wire.Envelope) {
			e := h.Entities[requirement]
			e.Fields[id(0xa0111)] = wire.Value{Tag: 7}
			h.Entities[requirement] = e
		},
		"parameter type": func(h *wire.Envelope) {
			e := h.Entities[requirement]
			e.Fields[id(0xa0111)] = wire.Value{Tag: 7, List: []wire.Value{{Tag: 6, Reference: other}}}
			h.Entities[requirement] = e
		},
		"result": func(h *wire.Envelope) {
			e := h.Entities[requirement]
			e.Fields[id(0xa0112)] = wire.Value{Tag: 6, Reference: other}
			h.Entities[requirement] = e
		},
		"receiver type": func(h *wire.Envelope) {
			e := h.Entities[receiver]
			e.Fields[id(0xa0001)] = wire.Value{Tag: 6, Reference: other}
			h.Entities[receiver] = e
		},
	}
	if err := validateMethodRequirement(g, method, requirement, concrete); err != nil {
		t.Fatalf("baseline: %v", err)
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			h := cloneEnvelope(g)
			mutate(&h)
			if err := validateMethodRequirement(h, method, requirement, concrete); err == nil {
				t.Fatal("forgery accepted")
			}
		})
	}
	if err := validateMethodRequirement(g, method, requirement, concrete); err != nil {
		t.Fatalf("baseline after mutations: %v", err)
	}
}

func TestInterfaceDispatchBudgetAndCycleAreBounded(t *testing.T) {
	x := id(0x8200)
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{x: {ID: x, Schema: id(0xa013), Fields: map[wire.ID]wire.Value{id(0xa0130): {Tag: 6, Reference: x}, id(0xa0131): {Tag: 6, Reference: x}, id(0xa0132): {Tag: 6, Reference: x}}}}}
	defer func() {
		if recover() != nil {
			t.Fatal("cycle panicked")
		}
	}()
	_, err := eval(g, x, map[wire.ID]Value{}, 3)
	if err == nil || (!strings.Contains(err.Error(), "interface") && !strings.Contains(err.Error(), "budget")) {
		t.Fatalf("cycle=%v", err)
	}
	_, err = eval(g, x, map[wire.ID]Value{}, 0)
	if err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("budget=%v", err)
	}
}
