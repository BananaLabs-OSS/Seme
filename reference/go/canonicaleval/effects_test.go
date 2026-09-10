package canonicaleval

import (
	"testing"

	"seme.local/reference/wire"
)

func TestEvaluateObservedAuthorizationOrderAndForgery(t *testing.T) {
	g := effectGraph()
	args := []Value{{Kind: "bool", Bool: true}, {Kind: "bool", Bool: false}}
	result, trace, err := EvaluateObserved(g, args, map[string]bool{"observability.log": true})
	if err != nil || result.Kind != "bool" || result.Bool || len(trace) != 2 || !trace[0].Value || trace[1].Value {
		t.Fatalf("ordered execution mismatch: result=%+v trace=%+v err=%v", result, trace, err)
	}
	if _, denied, err := EvaluateObserved(g, args, nil); err == nil || len(denied) != 0 {
		t.Fatalf("denial emitted or succeeded: trace=%+v err=%v", denied, err)
	}
	for name, mutate := range map[string]func(*wire.Envelope){
		"capability-reference": func(x *wire.Envelope) {
			e := x.Entities[id(20)]
			e.Fields[id(0x151)] = wire.Value{Tag: 6, Reference: id(99)}
			x.Entities[id(20)] = e
		},
		"effect-name": func(x *wire.Envelope) {
			e := x.Entities[id(20)]
			e.Fields[id(0x150)] = wire.Value{Tag: 5, Bytes: []byte("forged")}
			x.Entities[id(20)] = e
		},
		"capability-name": func(x *wire.Envelope) {
			e := x.Entities[id(21)]
			e.Fields[id(0x160)] = wire.Value{Tag: 5, Bytes: []byte("forged")}
			x.Entities[id(21)] = e
		},
		"argument-kind": func(x *wire.Envelope) {
			e := x.Entities[id(30)]
			e.Fields[id(0x9f11)] = wire.Value{Tag: 7, List: []wire.Value{{Tag: 5, Bytes: []byte("bad")}}}
			x.Entities[id(30)] = e
		},
		"late-effect-forgery": func(x *wire.Envelope) {
			e := x.Entities[id(31)]
			e.Fields[id(0x9f10)] = wire.Value{Tag: 6, Reference: id(99)}
			x.Entities[id(31)] = e
		},
	} {
		t.Run(name, func(t *testing.T) {
			x := cloneEnvelope(g)
			mutate(&x)
			if _, observed, err := EvaluateObserved(x, args, map[string]bool{"observability.log": true}); err == nil || len(observed) != 0 {
				t.Fatalf("forgery accepted/emitted: trace=%+v err=%v", observed, err)
			}
		})
	}
}

func TestGenericEvaluatorExecutesAuthorizedEffects(t *testing.T) {
	g := effectGraph()
	args := []Value{{Kind: "bool", Bool: true}, {Kind: "bool", Bool: false}}
	result, trace, err := EvaluateAuthorized(g, args, map[string]bool{"observability.log": true})
	if err != nil || result.Kind != "bool" || result.Bool || len(trace) != 2 || !trace[0].Value || trace[1].Value {
		t.Fatalf("result=%#v trace=%#v err=%v", result, trace, err)
	}
	if _, trace, err := EvaluateAuthorized(g, args, nil); err == nil || len(trace) != 0 {
		t.Fatalf("denial trace=%#v err=%v", trace, err)
	}
}

func TestGenericEvaluatorIgnoresUnreachableEffectDeclarations(t *testing.T) {
	g := effectGraph()
	ref := func(n uint64) wire.Value { return wire.Value{Tag: 6, Reference: id(n)} }
	list := func(ns ...uint64) wire.Value {
		value := wire.Value{Tag: 7}
		for _, n := range ns {
			value.List = append(value.List, ref(n))
		}
		return value
	}
	g.Entities[id(40)] = wire.Entity{ID: id(40), Schema: id(0x9081), Fields: map[wire.ID]wire.Value{id(0x9810): list(4)}}
	g.Entities[id(41)] = wire.Entity{ID: id(41), Schema: id(0x9080), Fields: map[wire.ID]wire.Value{id(0x9800): list(40)}}
	g.Entities[id(42)] = wire.Entity{ID: id(42), Schema: id(0x9011), Fields: map[wire.ID]wire.Value{id(0x9111): list(2), id(0x9113): ref(41)}}
	program := g.Entities[id(35)]
	program.Fields[id(0x9151)] = ref(42)
	g.Entities[id(35)] = program
	result, trace, err := EvaluateAuthorized(g, []Value{{Kind: "bool", Bool: true}}, nil)
	if err != nil || !result.Bool || len(trace) != 0 {
		t.Fatalf("result=%#v trace=%#v err=%v", result, trace, err)
	}
}

func TestReturnedBlockTypeAllowsPriorNonReturnStatement(t *testing.T) {
	ref := func(n uint64) wire.Value { return wire.Value{Tag: 6, Reference: id(n)} }
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	g.Entities[id(1)] = wire.Entity{ID: id(1), Schema: id(0x9010)}
	g.Entities[id(2)] = wire.Entity{ID: id(2), Schema: id(0x9070), Fields: map[wire.ID]wire.Value{id(0x9701): ref(1)}}
	g.Entities[id(3)] = wire.Entity{ID: id(3), Schema: id(0x90d1)}
	g.Entities[id(4)] = wire.Entity{ID: id(4), Schema: id(0x9081), Fields: map[wire.ID]wire.Value{id(0x9810): {Tag: 7, List: []wire.Value{ref(2)}}}}
	g.Entities[id(5)] = wire.Entity{ID: id(5), Schema: id(0x9080), Fields: map[wire.ID]wire.Value{id(0x9800): {Tag: 7, List: []wire.Value{ref(3), ref(4)}}}}
	got, ok := returnedBlockType(g, id(5))
	if !ok || got != id(1) {
		t.Fatalf("type=%s ok=%t", got.String(), ok)
	}
}

func effectGraph() wire.Envelope {
	ref := func(n uint64) wire.Value { return wire.Value{Tag: 6, Reference: id(n)} }
	list := func(ns ...uint64) wire.Value {
		v := wire.Value{Tag: 7}
		for _, n := range ns {
			v.List = append(v.List, ref(n))
		}
		return v
	}
	e := map[wire.ID]wire.Entity{}
	put := func(n, schema uint64, fields map[wire.ID]wire.Value) {
		e[id(n)] = wire.Entity{ID: id(n), Schema: id(schema), Fields: fields}
	}
	put(1, 0x9020, map[wire.ID]wire.Value{})
	put(2, 0x9012, map[wire.ID]wire.Value{id(0x9121): ref(1)})
	put(3, 0x9012, map[wire.ID]wire.Value{id(0x9121): ref(1)})
	put(4, 0x9013, map[wire.ID]wire.Value{id(0x9130): ref(2)})
	put(5, 0x9013, map[wire.ID]wire.Value{id(0x9130): ref(3)})
	put(21, 0x16, map[wire.ID]wire.Value{id(0x160): {Tag: 5, Bytes: []byte("observability.log")}})
	put(20, 0x15, map[wire.ID]wire.Value{id(0x150): {Tag: 5, Bytes: []byte("observability.log")}, id(0x151): ref(21)})
	put(30, 0x90f1, map[wire.ID]wire.Value{id(0x9f10): ref(20), id(0x9f11): list(4)})
	put(31, 0x90f1, map[wire.ID]wire.Value{id(0x9f10): ref(20), id(0x9f11): list(5)})
	put(32, 0x9081, map[wire.ID]wire.Value{id(0x9810): list(5)})
	put(33, 0x9080, map[wire.ID]wire.Value{id(0x9800): list(30, 31, 32)})
	put(34, 0x9011, map[wire.ID]wire.Value{id(0x9111): list(2, 3), id(0x9113): ref(33)})
	put(35, 0x9015, map[wire.ID]wire.Value{id(0x9151): ref(34)})
	return wire.Envelope{Entities: e}
}
