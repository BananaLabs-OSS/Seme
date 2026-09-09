package canonicaleval

import (
	"fmt"
	"strings"
	"testing"

	"seme.local/reference/wire"
)

func TestAuditedScalarSemantics(t *testing.T) {
	v, err := add(Value{Kind: "i64", I64: "9223372036854775807"}, Value{Kind: "i64", I64: "1"})
	if err != nil || v.I64 != "-9223372036854775808" {
		t.Fatalf("wrap=%#v %v", v, err)
	}
	equal, err := canonicalBytesEqual("6F6B", "6f6b")
	if err != nil || !equal {
		t.Fatalf("decoded equality=%v %v", equal, err)
	}
	if _, err = canonicalBytesEqual("xyz", "00"); err == nil {
		t.Fatal("malformed hex accepted")
	}
}

func TestValidateValueCoversEverySupportedFamily(t *testing.T) {
	g, types := typeGraph()
	i := func(v string) Value { return Value{Kind: "i64", I64: v} }
	cases := []struct {
		name  string
		typ   wire.ID
		value Value
	}{
		{"i64", types["i64"], i("-1")}, {"bool", types["bool"], Value{Kind: "bool", Bool: true}},
		{"text", types["text"], Value{Kind: "text", Text: "世界"}}, {"bytes", types["bytes"], Value{Kind: "bytes", Bytes: "ff00"}},
		{"record", types["record"], Value{Kind: "record", Fields: map[string]Value{"value": i("1")}}},
		{"array", types["array"], Value{Kind: "array", Items: []Value{i("1"), i("2")}}},
		{"slice", types["slice"], Value{Kind: "slice", Items: []Value{i("1")}}},
		{"map", types["map"], Value{Kind: "map", ValueType: "i64", Entries: []Entry{{Key: i("-1"), Value: i("2")}}}},
		{"option", types["option"], Value{Kind: "option", Variant: "some", Payload: ptr(i("1"))}},
		{"result", types["result"], Value{Kind: "result", Variant: "error", Payload: ptr(Value{Kind: "text", Text: "bad"})}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateValue(g, tc.typ, tc.value, 32); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestValidateValueRejectsMalformedAndTypeErasure(t *testing.T) {
	g, types := typeGraph()
	i := func(v string) Value { return Value{Kind: "i64", I64: v} }
	cases := []struct {
		name  string
		typ   wire.ID
		value Value
	}{
		{"overflow", types["i64"], i("9223372036854775808")}, {"invalid bytes", types["bytes"], Value{Kind: "bytes", Bytes: "x"}},
		{"missing record field", types["record"], Value{Kind: "record", Fields: map[string]Value{}}},
		{"extra record field", types["record"], Value{Kind: "record", Fields: map[string]Value{"value": i("1"), "extra": i("2")}}},
		{"short array", types["array"], Value{Kind: "array", Items: []Value{i("1")}}},
		{"map value descriptor", types["map"], Value{Kind: "map", ValueType: "text"}},
		{"map order", types["map"], Value{Kind: "map", ValueType: "i64", Entries: []Entry{{Key: i("2"), Value: i("0")}, {Key: i("1"), Value: i("0")}}}},
		{"option payload", types["option"], Value{Kind: "option", Variant: "some"}},
		{"result variant", types["result"], Value{Kind: "result", Variant: "wat", Payload: ptr(i("0"))}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateValue(g, tc.typ, tc.value, 32); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	if err := validateValue(g, types["option"], Value{Kind: "option", Variant: "none"}, 0); err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("budget=%v", err)
	}
}

func TestEvaluateMalformedGraphsRejectWithoutPanicking(t *testing.T) {
	g := scalarGraph()
	arg := []Value{{Kind: "i64", I64: "1"}}
	mutations := []func(*wire.Envelope){
		func(g *wire.Envelope) {
			e := g.Entities[id(0x6002)]
			delete(e.Fields, id(0x9113))
			g.Entities[e.ID] = e
		},
		func(g *wire.Envelope) { e := g.Entities[id(0x6003)]; e.Schema = id(0x9031); g.Entities[e.ID] = e },
		func(g *wire.Envelope) {
			e := g.Entities[id(0x6007)]
			e.Fields[id(0x9130)] = wire.Value{Tag: 3, Unsigned: 1}
			g.Entities[e.ID] = e
		},
		func(g *wire.Envelope) {
			e := g.Entities[id(0x6006)]
			e.Fields[id(0x9810)] = wire.Value{Tag: 6, Reference: id(0x6007)}
			g.Entities[e.ID] = e
		},
	}
	for index, mutate := range mutations {
		copy := cloneGraph(g)
		mutate(&copy)
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			defer func() {
				if recover() != nil {
					t.Fatal("panic")
				}
			}()
			if _, err := Evaluate(copy, arg); err == nil {
				t.Fatal("malformed graph accepted")
			}
		})
	}
}

func TestEverySupportedExpressionRejectsMissingRequiredFieldWithoutPanic(t *testing.T) {
	cases := []struct{ schema, field uint64 }{
		{0x9013, 0x9130}, {0x9070, 0x9700}, {0x9050, 0x9500}, {0xa064, 0xa0640}, {0x90b0, 0x9b00},
		{0x9014, 0x9140}, {0x9021, 0x9160}, {0x90b1, 0x9b10}, {0x90c3, 0x9c30}, {0x9032, 0x9320},
		{0x90f4, 0x9f40}, {0x90f9, 0x9f90}, {0xa042, 0xa0420}, {0xa061, 0xa0610}, {0xa063, 0xa0630},
		{0xa062, 0xa0620}, {0xa065, 0xa0650}, {0x90c2, 0x9c20},
	}
	for _, tc := range cases {
		for _, wrong := range []bool{false, true} {
			t.Run(fmt.Sprintf("%x/wrong=%v", tc.schema, wrong), func(t *testing.T) {
				x := id(0x7100 + tc.schema)
				fields := map[wire.ID]wire.Value{}
				if wrong {
					fields[id(tc.field)] = wire.Value{Tag: 0}
				}
				g := wire.Envelope{Entities: map[wire.ID]wire.Entity{x: {ID: x, Schema: id(tc.schema), Fields: fields}}}
				defer func() {
					if recover() != nil {
						t.Fatal("panic")
					}
				}()
				_, err := eval(g, x, map[wire.ID]Value{}, 8)
				if err == nil || !strings.HasPrefix(err.Error(), "canonicaleval.") {
					t.Fatalf("error=%v", err)
				}
			})
		}
	}
}

func TestSelfReferentialExpressionIsBudgetBounded(t *testing.T) {
	x := id(0x7200)
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{x: {ID: x, Schema: id(0x9014), Fields: map[wire.ID]wire.Value{id(0x9140): {Tag: 6, Reference: x}, id(0x9141): {Tag: 6, Reference: x}}}}}
	defer func() {
		if recover() != nil {
			t.Fatal("panic")
		}
	}()
	_, err := eval(g, x, map[wire.ID]Value{}, 4)
	if err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("error=%v", err)
	}
}

func TestForgedRecordMemberAndVariantBindingReject(t *testing.T) {
	g, types := typeGraph()
	param, read, foreign, expression := id(0x7300), id(0x7301), id(0x7302), id(0x7303)
	g.Entities[param] = wire.Entity{ID: param, Schema: id(0x9012), Fields: map[wire.ID]wire.Value{id(0x9121): {Tag: 6, Reference: types["record"]}}}
	g.Entities[read] = wire.Entity{ID: read, Schema: id(0x9013), Fields: map[wire.ID]wire.Value{id(0x9130): {Tag: 6, Reference: param}}}
	g.Entities[foreign] = wire.Entity{ID: foreign, Schema: id(0x9031), Fields: map[wire.ID]wire.Value{id(0x9310): {Tag: 5, Bytes: []byte("value")}, id(0x9311): {Tag: 6, Reference: types["i64"]}}}
	g.Entities[expression] = wire.Entity{ID: expression, Schema: id(0x9032), Fields: map[wire.ID]wire.Value{id(0x9320): {Tag: 6, Reference: read}, id(0x9321): {Tag: 6, Reference: foreign}}}
	if _, err := eval(g, expression, map[wire.ID]Value{param: {Kind: "record", Fields: map[string]Value{"value": {Kind: "i64", I64: "1"}}}}, 8); err == nil || !strings.Contains(err.Error(), "record_member") {
		t.Fatalf("forged member=%v", err)
	}
	variant := id(0x7310)
	g.Entities[variant] = wire.Entity{ID: variant, Schema: id(0xa061), Fields: map[wire.ID]wire.Value{id(0xa0610): {Tag: 6, Reference: param}}}
	if _, err := eval(g, variant, map[wire.ID]Value{param: {Kind: "i64", I64: "1"}}, 8); err == nil || !strings.Contains(err.Error(), "unbound_variant") {
		t.Fatalf("forged binding=%v", err)
	}
}

func scalarGraph() wire.Envelope {
	typeID, program, fn, param, block, ret, read := id(0x6001), id(0x6000), id(0x6002), id(0x6003), id(0x6004), id(0x6006), id(0x6007)
	e := map[wire.ID]wire.Entity{}
	add := func(x, s wire.ID, f map[wire.ID]wire.Value) { e[x] = wire.Entity{ID: x, Schema: s, Fields: f} }
	add(typeID, id(0x9010), nil)
	add(program, id(0x9015), map[wire.ID]wire.Value{id(0x9151): {Tag: 6, Reference: fn}})
	add(fn, id(0x9011), map[wire.ID]wire.Value{id(0x9111): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: param}}}, id(0x9113): {Tag: 6, Reference: block}})
	add(param, id(0x9012), map[wire.ID]wire.Value{id(0x9121): {Tag: 6, Reference: typeID}})
	add(block, id(0x9080), map[wire.ID]wire.Value{id(0x9800): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: ret}}}})
	add(ret, id(0x9081), map[wire.ID]wire.Value{id(0x9810): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: read}}}})
	add(read, id(0x9013), map[wire.ID]wire.Value{id(0x9130): {Tag: 6, Reference: param}})
	return wire.Envelope{Entities: e}
}
func cloneGraph(g wire.Envelope) wire.Envelope {
	out := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for k, e := range g.Entities {
		fields := map[wire.ID]wire.Value{}
		for fk, v := range e.Fields {
			fields[fk] = v
		}
		e.Fields = fields
		out.Entities[k] = e
	}
	return out
}

func typeGraph() (wire.Envelope, map[string]wire.ID) {
	t := map[string]wire.ID{"i64": id(0x5001), "bool": id(0x5002), "text": id(0x5003), "bytes": id(0x5004), "record": id(0x5005), "field": id(0x5006), "array": id(0x5007), "slice": id(0x5008), "map": id(0x5009), "option": id(0x500a), "result": id(0x500b)}
	e := map[wire.ID]wire.Entity{}
	add := func(x, s wire.ID, f map[wire.ID]wire.Value) { e[x] = wire.Entity{ID: x, Schema: s, Fields: f} }
	add(t["i64"], id(0x9010), nil)
	add(t["bool"], id(0x9020), nil)
	add(t["text"], id(0x9040), nil)
	add(t["bytes"], id(0x9041), nil)
	add(t["field"], id(0x9031), map[wire.ID]wire.Value{id(0x9310): {Tag: 5, Bytes: []byte("value")}, id(0x9311): {Tag: 6, Reference: t["i64"]}})
	add(t["record"], id(0x9030), map[wire.ID]wire.Value{id(0x9301): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: t["field"]}}}})
	add(t["array"], id(0x90f2), map[wire.ID]wire.Value{id(0x9f20): {Tag: 6, Reference: t["i64"]}, id(0x9f21): {Tag: 3, Unsigned: 2}})
	add(t["slice"], id(0x90f8), map[wire.ID]wire.Value{id(0x9f80): {Tag: 6, Reference: t["i64"]}})
	add(t["map"], id(0xa040), map[wire.ID]wire.Value{id(0xa0400): {Tag: 6, Reference: t["i64"]}, id(0xa0401): {Tag: 6, Reference: t["i64"]}})
	add(t["option"], id(0xa050), map[wire.ID]wire.Value{id(0xa0500): {Tag: 6, Reference: t["i64"]}})
	add(t["result"], id(0x9042), map[wire.ID]wire.Value{id(0x9400): {Tag: 6, Reference: t["i64"]}, id(0x9401): {Tag: 6, Reference: t["text"]}})
	return wire.Envelope{Entities: e}, t
}
func ptr(v Value) *Value { return &v }
