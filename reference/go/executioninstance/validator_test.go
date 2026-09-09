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
func valid() wire.Envelope {
	typ, param, body, function, program := id(0x8001), id(0x8002), id(0x8003), id(0x8004), id(0x8005)
	return wire.Envelope{Module: executionModule, Revision: id(0x8123), Entities: map[wire.ID]wire.Entity{
		typ:      {ID: typ, Schema: id(0x9010), Version: 1, Fields: map[wire.ID]wire.Value{id(0x9100): {Tag: 4, Unsigned: 64}, id(0x9101): {Tag: 2}, id(0x9102): {Tag: 4}}},
		param:    {ID: param, Schema: id(0x9012), Version: 1, Fields: map[wire.ID]wire.Value{id(0x9120): {Tag: 5, Bytes: []byte("value")}, id(0x9121): {Tag: 6, Reference: typ}, id(0x9122): {Tag: 4}}},
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
			p.Fields[id(0x9122)] = wire.Value{Tag: 4, Unsigned: 1}
			e.Entities[p.ID] = p
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
