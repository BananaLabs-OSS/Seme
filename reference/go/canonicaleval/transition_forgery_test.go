package canonicaleval

import (
	"encoding/json"
	"strings"
	"testing"

	"seme.local/reference/wire"
)

func transitionGraph() (wire.Envelope, wire.ID) {
	ref := func(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
	list := func(xs ...wire.ID) wire.Value {
		values := make([]wire.Value, len(xs))
		for i, x := range xs {
			values[i] = ref(x)
		}
		return wire.Value{Tag: 7, List: values}
	}
	i64, member, record, transitionType := id(0x7a00), id(0x7a01), id(0x7a02), id(0x7a03)
	stateValue, resultValue, recordValue, transition := id(0x7a04), id(0x7a05), id(0x7a06), id(0x7a07)
	return wire.Envelope{Entities: map[wire.ID]wire.Entity{
		i64:            {ID: i64, Schema: id(0x9010)},
		member:         {ID: member, Schema: id(0x9031), Fields: map[wire.ID]wire.Value{id(0x9310): {Tag: 5, Bytes: []byte("value")}, id(0x9311): ref(i64), id(0x9312): {Tag: 3}}},
		record:         {ID: record, Schema: id(0x9030), Fields: map[wire.ID]wire.Value{id(0x9300): {Tag: 5, Bytes: []byte("Counter")}, id(0x9301): list(member)}},
		transitionType: {ID: transitionType, Schema: id(0xa004), Fields: map[wire.ID]wire.Value{id(0xa0040): ref(record), id(0xa0041): ref(i64)}},
		stateValue:     {ID: stateValue, Schema: id(0x9070), Fields: map[wire.ID]wire.Value{id(0x9700): {Tag: 3, Unsigned: 5}}},
		resultValue:    {ID: resultValue, Schema: id(0x9070), Fields: map[wire.ID]wire.Value{id(0x9700): {Tag: 3, Unsigned: ^uint64(6)}}},
		recordValue:    {ID: recordValue, Schema: id(0x9033), Fields: map[wire.ID]wire.Value{id(0x9330): ref(record), id(0x9331): list(stateValue)}},
		transition:     {ID: transition, Schema: id(0xa005), Fields: map[wire.ID]wire.Value{id(0xa0050): ref(transitionType), id(0xa0051): ref(recordValue), id(0xa0052): ref(resultValue)}},
	}}, transition
}

func TestGenericTransitionEvaluatesAndExportsBothValues(t *testing.T) {
	g, transition := transitionGraph()
	value, err := eval(g, transition, map[wire.ID]Value{}, 16)
	if err != nil {
		t.Fatal(err)
	}
	if value.Kind != "transition" || value.State == nil || value.Result == nil || value.State.Fields["value"].I64 != "5" || value.Result.I64 != "-7" {
		t.Fatalf("value=%#v", value)
	}
	typeID, ok := expressionType(g, transition)
	if !ok || validateType(g, typeID, map[wire.ID]bool{}, 16) != nil || validateValue(g, typeID, value, 16) != nil {
		t.Fatal("typed transition did not validate")
	}
	encoded, err := json.Marshal(value)
	if err != nil || !strings.Contains(string(encoded), `"state"`) || !strings.Contains(string(encoded), `"result"`) {
		t.Fatalf("json=%s err=%v", encoded, err)
	}
}

func TestGenericTransitionRejectsTypeAndGraphForgeries(t *testing.T) {
	g, transition := transitionGraph()
	transitionType := g.Entities[transition].Fields[id(0xa0050)].Reference
	for name, mutate := range map[string]func(*wire.Envelope){
		"missing-type": func(g *wire.Envelope) {
			e := g.Entities[transition]
			delete(e.Fields, id(0xa0050))
			g.Entities[transition] = e
		},
		"non-transition-type": func(g *wire.Envelope) {
			e := g.Entities[transition]
			e.Fields[id(0xa0050)] = wire.Value{Tag: 6, Reference: id(0x7a00)}
			g.Entities[transition] = e
		},
		"wrong-state": func(g *wire.Envelope) {
			e := g.Entities[transition]
			e.Fields[id(0xa0051)] = e.Fields[id(0xa0052)]
			g.Entities[transition] = e
		},
		"wrong-result": func(g *wire.Envelope) {
			e := g.Entities[transition]
			e.Fields[id(0xa0052)] = e.Fields[id(0xa0051)]
			g.Entities[transition] = e
		},
		"cycle": func(g *wire.Envelope) {
			e := g.Entities[transition]
			e.Fields[id(0xa0051)] = wire.Value{Tag: 6, Reference: transition}
			g.Entities[transition] = e
		},
	} {
		t.Run(name, func(t *testing.T) {
			forged := cloneGraph(g)
			mutate(&forged)
			if _, err := eval(forged, transition, map[wire.ID]Value{}, 8); err == nil {
				t.Fatal("forgery accepted")
			}
		})
	}
	cyclicType := cloneGraph(g)
	e := cyclicType.Entities[transitionType]
	e.Fields[id(0xa0040)] = wire.Value{Tag: 6, Reference: transitionType}
	cyclicType.Entities[transitionType] = e
	if err := validateType(cyclicType, transitionType, map[wire.ID]bool{}, 8); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("cyclic type=%v", err)
	}
}
