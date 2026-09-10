package executioninstance

import (
	"os"
	"strings"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/wire"
)

func contract(t *testing.T) contractcatalog.Contract {
	t.Helper()
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	set, e := contractcatalog.ResolveProjectContractSet(read("../../../modules/execution/v35/module.seme"), read("../../../modules/package/v1/module.seme"), read("../../../modules/project/v1/module.seme"))
	if e != nil {
		t.Fatal(e)
	}
	return set.Execution()
}
func contractV36(t *testing.T) contractcatalog.Contract {
	t.Helper()
	read := func(p string) []byte {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	set, err := contractcatalog.ResolveProjectContractSetV8(read("../../../modules/foundation/v1/module.seme"), read("../../../modules/execution/v36/module.seme"), read("../../../modules/package/v4/module.seme"), read("../../../modules/dependency/v1/module.seme"), read("../../../modules/configuration/v3/module.seme"), read("../../../modules/project/v8/module.seme"))
	if err != nil {
		t.Fatal(err)
	}
	return set.Execution()
}
func valid() wire.Envelope {
	typ, param, body, function, program := id(0x8001), id(0x8002), id(0x8003), id(0x8004), id(0x8005)
	return wire.Envelope{Module: executionModule, Revision: id(0x8123), Entities: map[wire.ID]wire.Entity{
		typ:      {ID: typ, Schema: id(0x9010), Version: 1, Fields: map[wire.ID]wire.Value{id(0x9100): {Tag: 3, Unsigned: 64}, id(0x9101): {Tag: 2}, id(0x9102): {Tag: 3}}},
		param:    {ID: param, Schema: id(0x9012), Version: 1, Fields: map[wire.ID]wire.Value{id(0x9120): {Tag: 5, Bytes: []byte("value")}, id(0x9121): {Tag: 6, Reference: typ}, id(0x9122): {Tag: 3}}},
		body:     {ID: body, Schema: id(0x9013), Version: 1, Fields: map[wire.ID]wire.Value{id(0x9130): {Tag: 6, Reference: param}}},
		function: {ID: function, Schema: id(0x9011), Version: 1, Fields: map[wire.ID]wire.Value{id(0x9110): {Tag: 5, Bytes: []byte("Apply")}, id(0x9111): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: param}}}, id(0x9112): {Tag: 6, Reference: typ}, id(0x9113): {Tag: 6, Reference: body}}},
		program:  {ID: program, Schema: id(0x9015), Version: 1, Fields: map[wire.ID]wire.Value{id(0x9150): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: function}}}, id(0x9151): {Tag: 6, Reference: function}}},
	}}
}
func clone(e wire.Envelope) wire.Envelope { b, _ := wire.Encode(e); x, _ := wire.Decode(b); return x }
func TestValidateExecutionV35Instance(t *testing.T) {
	if e := Validate(contract(t), valid()); e != nil {
		t.Fatal(e)
	}
}

func TestValidateExecutionV36AuthorityIsAdditive(t *testing.T) {
	e := valid()
	booleanType, literal, not := id(0x8100), id(0x8101), id(0x8102)
	e.Entities[booleanType] = wire.Entity{ID: booleanType, Schema: id(0x9020), Version: 1, Fields: map[wire.ID]wire.Value{}}
	e.Entities[literal] = wire.Entity{ID: literal, Schema: id(0x90b0), Version: 1, Fields: map[wire.ID]wire.Value{id(0x9b00): {Tag: 1, Unsigned: 1}}}
	e.Entities[not] = wire.Entity{ID: not, Schema: id(0xa069), Version: 1, Fields: map[wire.ID]wire.Value{id(0xa0690): {Tag: 6, Reference: literal}}}
	fn := e.Entities[id(0x8004)]
	fn.Fields[id(0x9112)] = wire.Value{Tag: 6, Reference: booleanType}
	fn.Fields[id(0x9113)] = wire.Value{Tag: 6, Reference: not}
	e.Entities[fn.ID] = fn
	delete(e.Entities, id(0x8003))
	if err := ValidateV36(contractV36(t), e); err != nil {
		t.Fatal(err)
	}
	if err := Validate(contract(t), e); err == nil {
		t.Fatal("Execution v35 accepted v36 BooleanNot")
	}
	if err := ValidateV36(contract(t), e); err == nil {
		t.Fatal("v36 validator accepted v35 authority")
	}
}
func TestFoundationEffectAndCapabilityAreStrictRuntimeBoundaryValues(t *testing.T) {
	base := valid()
	effect, capability, invoke := id(0x8201), id(0x8202), id(0x8203)
	base.Entities[capability] = wire.Entity{ID: capability, Schema: id(0x16), Version: 1, Fields: map[wire.ID]wire.Value{id(0x160): {Tag: 5, Bytes: []byte("observability.log")}}}
	base.Entities[effect] = wire.Entity{ID: effect, Schema: id(0x15), Version: 1, Fields: map[wire.ID]wire.Value{id(0x150): {Tag: 5, Bytes: []byte("observability.log")}, id(0x151): {Tag: 6, Reference: capability}}}
	base.Entities[invoke] = wire.Entity{ID: invoke, Schema: id(0x90f1), Version: 1, Fields: map[wire.ID]wire.Value{id(0x9f10): {Tag: 6, Reference: effect}, id(0x9f11): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: id(0x8003)}}}}}
	function := base.Entities[id(0x8004)]
	function.Fields[id(0x9113)] = wire.Value{Tag: 6, Reference: invoke}
	base.Entities[function.ID] = function
	if err := Validate(contract(t), base); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*wire.Envelope){
		"effect-extra-field": func(e *wire.Envelope) {
			q := e.Entities[effect]
			q.Fields[id(0x999)] = wire.Value{Tag: 5}
			e.Entities[effect] = q
		},
		"capability-wrong-schema":   func(e *wire.Envelope) { q := e.Entities[capability]; q.Schema = id(0x17); e.Entities[capability] = q },
		"unknown-foundation-schema": func(e *wire.Envelope) { q := e.Entities[capability]; q.Schema = id(0x18); e.Entities[capability] = q },
	} {
		t.Run(name, func(t *testing.T) {
			forged := clone(base)
			mutate(&forged)
			if err := Validate(contract(t), forged); err == nil {
				t.Fatal("forged Foundation runtime value accepted")
			}
		})
	}
}
func TestRejectsMalformedExecutionClosure(t *testing.T) {
	c := contract(t)
	base := valid()
	tests := map[string]func(wire.Envelope){
		"duplicate-functions": func(e wire.Envelope) {
			p := e.Entities[id(0x8005)]
			v := p.Fields[id(0x9150)]
			v.List = append(v.List, v.List[0])
			p.Fields[id(0x9150)] = v
			e.Entities[p.ID] = p
		},
		"parameter-index": func(e wire.Envelope) {
			p := e.Entities[id(0x8002)]
			p.Fields[id(0x9122)] = wire.Value{Tag: 3, Unsigned: 1}
			e.Entities[p.ID] = p
		},
		"signed-for-unsigned-field": func(e wire.Envelope) {
			typ := e.Entities[id(0x8001)]
			typ.Fields[id(0x9100)] = wire.Value{Tag: 4, Unsigned: 64}
			e.Entities[typ.ID] = typ
		},
		"missing-body": func(e wire.Envelope) { f := e.Entities[id(0x8004)]; delete(f.Fields, id(0x9113)); e.Entities[f.ID] = f },
		"unknown-field": func(e wire.Envelope) {
			f := e.Entities[id(0x8004)]
			f.Fields[id(0xffff)] = wire.Value{}
			e.Entities[f.ID] = f
		},
		"wrong-type-schema": func(e wire.Envelope) {
			f := e.Entities[id(0x8004)]
			f.Fields[id(0x9111)] = wire.Value{Tag: 7, List: []wire.Value{{Tag: 6, Reference: id(0x8001)}}}
			e.Entities[f.ID] = f
		},
		"unreachable": func(e wire.Envelope) {
			x := id(0x8999)
			e.Entities[x] = wire.Entity{ID: x, Schema: id(0x9020), Version: 1, Fields: map[wire.ID]wire.Value{}}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			e := clone(base)
			mutate(e)
			if err := Validate(c, e); err == nil {
				t.Fatal("accepted")
			}
		})
	}
}
func TestRejectsEntryOutsideFunctionSetAndWrongContract(t *testing.T) {
	e := valid()
	other := id(0x8010)
	e.Entities[other] = e.Entities[id(0x8004)]
	x := e.Entities[other]
	x.ID = other
	e.Entities[other] = x
	p := e.Entities[id(0x8005)]
	p.Fields[id(0x9151)] = wire.Value{Tag: 6, Reference: other}
	e.Entities[p.ID] = p
	if err := Validate(contract(t), e); err == nil || !strings.Contains(err.Error(), "entry_membership") {
		t.Fatalf("got %v", err)
	}
	if err := Validate(contractcatalog.Contract{}, valid()); err == nil {
		t.Fatal("accepted unchecked contract")
	}
}
