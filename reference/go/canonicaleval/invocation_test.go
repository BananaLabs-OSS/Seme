package canonicaleval

import (
	"testing"

	"seme.local/reference/wire"
)

func TestEvaluateFunctionDirectWithExactTypedArguments(t *testing.T) {
	g := invocationGraph()
	got, err := EvaluateFunction(g, id(0x7002), []Value{{Kind: "i64", I64: "42"}})
	if err != nil || got.Kind != "i64" || got.I64 != "42" {
		t.Fatalf("got=%#v err=%v", got, err)
	}
	for name, run := range map[string]func() error{
		"non-function": func() error { _, err := EvaluateFunction(g, id(0x7001), []Value{{Kind: "i64", I64: "1"}}); return err },
		"wrong-arity":  func() error { _, err := EvaluateFunction(g, id(0x7002), nil); return err },
		"wrong-type": func() error {
			_, err := EvaluateFunction(g, id(0x7002), []Value{{Kind: "bool", Bool: true}})
			return err
		},
		"closed-expression": func() error { _, err := EvaluateExpression(g, id(0x7007)); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if run() == nil {
				t.Fatal("forgery accepted")
			}
		})
	}
	bad := cloneGraph(g)
	p := bad.Entities[id(0x7003)]
	p.Fields[id(0x9122)] = wire.Value{Tag: 3, Unsigned: 1}
	bad.Entities[p.ID] = p
	if _, err = EvaluateFunction(bad, id(0x7002), []Value{{Kind: "i64", I64: "1"}}); err == nil {
		t.Fatal("parameter reorder accepted")
	}
	bad = cloneGraph(g)
	f := bad.Entities[id(0x7002)]
	f.Fields[id(0x9112)] = wire.Value{Tag: 6, Reference: id(0x7008)}
	bad.Entities[f.ID] = f
	if _, err = EvaluateFunction(bad, id(0x7002), []Value{{Kind: "i64", I64: "1"}}); err == nil {
		t.Fatal("wrong result accepted")
	}
}

func TestEvaluateExpressionClosedPureAndTyped(t *testing.T) {
	g := invocationGraph()
	got, err := EvaluateExpression(g, id(0x7009))
	if err != nil || got.Kind != "i64" || got.I64 != "-1" {
		t.Fatalf("got=%#v err=%v", got, err)
	}
	bad := cloneGraph(g)
	literal := bad.Entities[id(0x7009)]
	literal.Fields[id(0x9701)] = wire.Value{Tag: 6, Reference: id(0x7008)}
	bad.Entities[literal.ID] = literal
	if _, err = EvaluateExpression(bad, id(0x7009)); err == nil {
		t.Fatal("forged literal type accepted")
	}
}

func TestEvaluateFunctionEffectsRequireAuthorityBeforeTrace(t *testing.T) {
	g := effectGraph()
	for i, pid := range []wire.ID{id(2), id(3)} {
		p := g.Entities[pid]
		p.Fields[id(0x9122)] = wire.Value{Tag: 3, Unsigned: uint64(i)}
		g.Entities[pid] = p
	}
	fn := g.Entities[id(34)]
	fn.Fields[id(0x9112)] = wire.Value{Tag: 6, Reference: id(1)}
	g.Entities[fn.ID] = fn
	args := []Value{{Kind: "bool", Bool: true}, {Kind: "bool", Bool: false}}
	if _, trace, err := EvaluateFunctionAuthorized(g, id(34), args, nil); err == nil || len(trace) != 0 {
		t.Fatalf("denial trace=%#v err=%v", trace, err)
	}
	got, trace, err := EvaluateFunctionAuthorized(g, id(34), args, map[string]bool{"observability.log": true})
	if err != nil || got.Kind != "bool" || got.Bool || len(trace) != 2 {
		t.Fatalf("got=%#v trace=%#v err=%v", got, trace, err)
	}
}

func invocationGraph() wire.Envelope {
	i64, fn, param, block, ret, read, boolean, literal := id(0x7001), id(0x7002), id(0x7003), id(0x7004), id(0x7005), id(0x7006), id(0x7008), id(0x7009)
	e := map[wire.ID]wire.Entity{}
	e[i64] = wire.Entity{ID: i64, Schema: id(0x9010), Fields: map[wire.ID]wire.Value{}}
	e[boolean] = wire.Entity{ID: boolean, Schema: id(0x9020), Fields: map[wire.ID]wire.Value{}}
	e[param] = wire.Entity{ID: param, Schema: id(0x9012), Fields: map[wire.ID]wire.Value{id(0x9121): {Tag: 6, Reference: i64}, id(0x9122): {Tag: 3, Unsigned: 0}}}
	e[read] = wire.Entity{ID: read, Schema: id(0x9013), Fields: map[wire.ID]wire.Value{id(0x9130): {Tag: 6, Reference: param}}}
	e[ret] = wire.Entity{ID: ret, Schema: id(0x9081), Fields: map[wire.ID]wire.Value{id(0x9810): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: read}}}}}
	e[block] = wire.Entity{ID: block, Schema: id(0x9080), Fields: map[wire.ID]wire.Value{id(0x9800): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: ret}}}}}
	e[fn] = wire.Entity{ID: fn, Schema: id(0x9011), Fields: map[wire.ID]wire.Value{id(0x9111): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: param}}}, id(0x9112): {Tag: 6, Reference: i64}, id(0x9113): {Tag: 6, Reference: block}}}
	e[literal] = wire.Entity{ID: literal, Schema: id(0x9070), Fields: map[wire.ID]wire.Value{id(0x9700): {Tag: 3, Unsigned: ^uint64(0)}, id(0x9701): {Tag: 6, Reference: i64}}}
	return wire.Envelope{Entities: e}
}
