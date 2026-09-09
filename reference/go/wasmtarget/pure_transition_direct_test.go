package wasmtarget

import (
	"testing"

	"seme.local/reference/wire"
)

func directTransitionGraph() (wire.Envelope, wire.Entity, wire.Entity, wire.ID) {
	ref := func(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
	list := func(xs ...wire.ID) wire.Value {
		out := make([]wire.Value, len(xs))
		for i, x := range xs {
			out[i] = ref(x)
		}
		return wire.Value{Tag: 7, List: out}
	}
	i64, member, recordType, transitionType := identity(0x7b00), identity(0x7b01), identity(0x7b02), identity(0x7b03)
	stateParameter, deltaParameter := identity(0x7b04), identity(0x7b05)
	stateRead, deltaRead, oldValue, sum, nextState := identity(0x7b06), identity(0x7b07), identity(0x7b08), identity(0x7b09), identity(0x7b0a)
	transition, returned, block, functionID, programID := identity(0x7b0b), identity(0x7b0c), identity(0x7b0d), identity(0x7b0e), identity(0x7b0f)
	entities := map[wire.ID]wire.Entity{
		i64:            {ID: i64, Schema: identity(0x9010), Fields: map[wire.ID]wire.Value{identity(0x9100): {Tag: 3, Unsigned: 64}, identity(0x9101): {Tag: 2}, identity(0x9102): {Tag: 3}}},
		member:         {ID: member, Schema: identity(0x9031), Fields: map[wire.ID]wire.Value{identity(0x9310): {Tag: 5, Bytes: []byte("value")}, identity(0x9311): ref(i64), identity(0x9312): {Tag: 3}}},
		recordType:     {ID: recordType, Schema: identity(0x9030), Fields: map[wire.ID]wire.Value{identity(0x9300): {Tag: 5, Bytes: []byte("Counter")}, identity(0x9301): list(member)}},
		transitionType: {ID: transitionType, Schema: identity(0xa004), Fields: map[wire.ID]wire.Value{identity(0xa0040): ref(recordType), identity(0xa0041): ref(i64)}},
		stateParameter: {ID: stateParameter, Schema: identity(0x9012), Fields: map[wire.ID]wire.Value{identity(0x9120): {Tag: 5, Bytes: []byte("state")}, identity(0x9121): ref(recordType), identity(0x9122): {Tag: 3}}},
		deltaParameter: {ID: deltaParameter, Schema: identity(0x9012), Fields: map[wire.ID]wire.Value{identity(0x9120): {Tag: 5, Bytes: []byte("delta")}, identity(0x9121): ref(i64), identity(0x9122): {Tag: 3, Unsigned: 1}}},
		stateRead:      {ID: stateRead, Schema: identity(0x9013), Fields: map[wire.ID]wire.Value{identity(0x9130): ref(stateParameter)}},
		deltaRead:      {ID: deltaRead, Schema: identity(0x9013), Fields: map[wire.ID]wire.Value{identity(0x9130): ref(deltaParameter)}},
		oldValue:       {ID: oldValue, Schema: identity(0x9032), Fields: map[wire.ID]wire.Value{identity(0x9320): ref(stateRead), identity(0x9321): ref(member)}},
		sum:            {ID: sum, Schema: identity(0x9014), Fields: map[wire.ID]wire.Value{identity(0x9140): ref(oldValue), identity(0x9141): ref(deltaRead), identity(0x9142): ref(i64)}},
		nextState:      {ID: nextState, Schema: identity(0x9033), Fields: map[wire.ID]wire.Value{identity(0x9330): ref(recordType), identity(0x9331): list(sum)}},
		transition:     {ID: transition, Schema: identity(0xa005), Fields: map[wire.ID]wire.Value{identity(0xa0050): ref(transitionType), identity(0xa0051): ref(nextState), identity(0xa0052): ref(oldValue)}},
		returned:       {ID: returned, Schema: identity(0x9081), Fields: map[wire.ID]wire.Value{identity(0x9810): list(transition)}},
		block:          {ID: block, Schema: identity(0x9080), Fields: map[wire.ID]wire.Value{identity(0x9800): list(returned)}},
		functionID:     {ID: functionID, Schema: identity(0x9011), Fields: map[wire.ID]wire.Value{identity(0x9110): {Tag: 5, Bytes: []byte("Step")}, identity(0x9111): list(stateParameter, deltaParameter), identity(0x9112): ref(transitionType), identity(0x9113): ref(block)}},
		programID:      {ID: programID, Schema: identity(0x9015), Fields: map[wire.ID]wire.Value{identity(0x9150): list(functionID), identity(0x9151): ref(functionID)}},
	}
	g := wire.Envelope{Module: identity(0x7b10), Revision: identity(0x7b11), Entities: entities}
	return g, entities[programID], entities[functionID], transition
}

func TestDirectTransitionTargetAcceptsTypedValue(t *testing.T) {
	g, program, function, _ := directTransitionGraph()
	wasm, abi, err := certifyPureTransitionFunction(g, program, function)
	if err != nil {
		t.Fatal(err)
	}
	if len(wasm) == 0 || abi.RequestSize != 16 || abi.ResponseSize != 16 || abi.Result.Type != "state-transition:record:i64,i64" {
		t.Fatalf("abi=%#v wasm=%d", abi, len(wasm))
	}
}

func TestDirectTransitionTargetRejectsForgeries(t *testing.T) {
	g, program, function, transition := directTransitionGraph()
	for name, mutate := range map[string]func(*wire.Envelope){
		"wrong-type": func(g *wire.Envelope) {
			e := g.Entities[transition]
			e.Fields[identity(0xa0050)] = wire.Value{Tag: 6, Reference: identity(0x7b00)}
			g.Entities[transition] = e
		},
		"swapped-state-result": func(g *wire.Envelope) {
			e := g.Entities[transition]
			a, b := e.Fields[identity(0xa0051)], e.Fields[identity(0xa0052)]
			e.Fields[identity(0xa0051)], e.Fields[identity(0xa0052)] = b, a
			g.Entities[transition] = e
		},
		"cycle": func(g *wire.Envelope) {
			e := g.Entities[transition]
			e.Fields[identity(0xa0051)] = wire.Value{Tag: 6, Reference: transition}
			g.Entities[transition] = e
		},
	} {
		t.Run(name, func(t *testing.T) {
			forged := cloneWireGraph(g)
			mutate(&forged)
			if _, _, err := certifyPureTransitionFunction(forged, program, function); err == nil {
				t.Fatal("forgery accepted")
			}
		})
	}
}
