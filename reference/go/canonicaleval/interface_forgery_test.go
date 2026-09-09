package canonicaleval

import (
	"strings"
	"testing"

	"seme.local/reference/wire"
)

func TestInterfaceValueRejectsForgedConcreteAndInterfaceWitness(t *testing.T) {
	g, ids := interfaceForgeryGraph()
	if _, err := eval(g, ids["interfaceValue"], map[wire.ID]Value{}, 32); err != nil {
		t.Fatalf("valid interface value: %v", err)
	}
	for _, tc := range []struct {
		name  string
		field uint64
		value wire.ID
	}{
		{"concrete", 0xa0120, ids["recordB"]},
		{"interface", 0xa0121, ids["interfaceB"]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			forged := cloneGraph(g)
			w := forged.Entities[ids["witness"]]
			w.Fields[id(tc.field)] = wire.Value{Tag: 6, Reference: tc.value}
			forged.Entities[w.ID] = w
			if _, err := eval(forged, ids["interfaceValue"], map[wire.ID]Value{}, 32); err == nil || !strings.Contains(err.Error(), "interface_value") {
				t.Fatalf("forgery accepted: %v", err)
			}
		})
	}
}

func TestMethodRequirementRejectsForgedSignatureResultAndReceiverOwnership(t *testing.T) {
	g, ids := interfaceForgeryGraph()
	if err := validateMethodRequirement(g, ids["method"], ids["requirement"], ids["recordA"]); err != nil {
		t.Fatalf("valid method: %v", err)
	}
	mutations := map[string]func(*wire.Envelope){
		"name": func(g *wire.Envelope) {
			e := g.Entities[ids["method"]]
			e.Fields[id(0xa0020)] = wire.Value{Tag: 5, Bytes: []byte("Other")}
			g.Entities[e.ID] = e
		},
		"arity": func(g *wire.Envelope) {
			e := g.Entities[ids["method"]]
			e.Fields[id(0xa0022)] = wire.Value{Tag: 7}
			g.Entities[e.ID] = e
		},
		"parameter": func(g *wire.Envelope) {
			e := g.Entities[ids["parameter"]]
			e.Fields[id(0x9121)] = wire.Value{Tag: 6, Reference: ids["bool"]}
			g.Entities[e.ID] = e
		},
		"result": func(g *wire.Envelope) {
			e := g.Entities[ids["method"]]
			e.Fields[id(0xa0023)] = wire.Value{Tag: 6, Reference: ids["bool"]}
			g.Entities[e.ID] = e
		},
		"receiver ownership": func(g *wire.Envelope) {
			e := g.Entities[ids["receiver"]]
			e.Fields[id(0xa0001)] = wire.Value{Tag: 6, Reference: ids["recordB"]}
			g.Entities[e.ID] = e
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			forged := cloneGraph(g)
			mutate(&forged)
			if err := validateMethodRequirement(forged, ids["method"], ids["requirement"], ids["recordA"]); err == nil {
				t.Fatal("forgery accepted")
			}
		})
	}
}

func interfaceForgeryGraph() (wire.Envelope, map[string]wire.ID) {
	n := map[string]wire.ID{}
	for i, name := range []string{"i64", "bool", "fieldA", "fieldB", "recordA", "recordB", "interfaceA", "interfaceB", "requirement", "receiver", "parameter", "method", "witness", "literal", "construct", "interfaceValue", "body"} {
		n[name] = id(uint64(0x7800 + i))
	}
	e := map[wire.ID]wire.Entity{}
	add := func(name string, schema uint64, fields map[wire.ID]wire.Value) {
		x := n[name]
		e[x] = wire.Entity{ID: x, Schema: id(schema), Fields: fields}
	}
	ref := func(name string) wire.Value { return wire.Value{Tag: 6, Reference: n[name]} }
	refs := func(names ...string) wire.Value {
		values := make([]wire.Value, len(names))
		for i, name := range names {
			values[i] = ref(name)
		}
		return wire.Value{Tag: 7, List: values}
	}
	add("i64", 0x9010, nil)
	add("bool", 0x9020, nil)
	add("fieldA", 0x9031, map[wire.ID]wire.Value{id(0x9310): {Tag: 5, Bytes: []byte("Amount")}, id(0x9311): ref("i64")})
	add("fieldB", 0x9031, map[wire.ID]wire.Value{id(0x9310): {Tag: 5, Bytes: []byte("Amount")}, id(0x9311): ref("i64")})
	add("recordA", 0x9030, map[wire.ID]wire.Value{id(0x9301): refs("fieldA")})
	add("recordB", 0x9030, map[wire.ID]wire.Value{id(0x9301): refs("fieldB")})
	add("requirement", 0xa011, map[wire.ID]wire.Value{id(0xa0110): {Tag: 5, Bytes: []byte("Adjust")}, id(0xa0111): refs("i64"), id(0xa0112): ref("i64")})
	add("interfaceA", 0xa010, map[wire.ID]wire.Value{id(0xa0101): refs("requirement")})
	add("interfaceB", 0xa010, map[wire.ID]wire.Value{id(0xa0101): refs("requirement")})
	add("receiver", 0xa000, map[wire.ID]wire.Value{id(0xa0001): ref("recordA")})
	add("parameter", 0x9012, map[wire.ID]wire.Value{id(0x9121): ref("i64"), id(0x9122): {Tag: 3}})
	add("body", 0x9080, map[wire.ID]wire.Value{id(0x9800): {Tag: 7}})
	add("method", 0xa002, map[wire.ID]wire.Value{id(0xa0020): {Tag: 5, Bytes: []byte("Adjust")}, id(0xa0021): ref("receiver"), id(0xa0022): refs("parameter"), id(0xa0023): ref("i64"), id(0xa0024): ref("body")})
	add("witness", 0xa012, map[wire.ID]wire.Value{id(0xa0120): ref("recordA"), id(0xa0121): ref("interfaceA"), id(0xa0122): refs("method")})
	add("literal", 0x9070, map[wire.ID]wire.Value{id(0x9700): {Tag: 3, Unsigned: 3}})
	add("construct", 0x9033, map[wire.ID]wire.Value{id(0x9330): ref("recordA"), id(0x9331): refs("literal")})
	add("interfaceValue", 0xa013, map[wire.ID]wire.Value{id(0xa0130): ref("interfaceA"), id(0xa0131): ref("construct"), id(0xa0132): ref("witness")})
	return wire.Envelope{Entities: e}, n
}
