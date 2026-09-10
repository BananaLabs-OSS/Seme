package goconfigurationadapter

import (
	"fmt"
	"testing"

	"seme.local/reference/wire"
)

func TestResolveBuildsExactThreeStageBoundPlan(t *testing.T) {
	i64, text, boolean := id("1"), id("2"), id("3")
	input, configured, prepared, state, runtime := id("100"), id("101"), id("102"), id("103"), id("104")
	configFn, policyFn, assembleFn := id("200"), id("201"), id("202")
	nameCap, limitCap, defaultCap := id("210"), id("211"), id("212")
	configOwner, policyOwner, serviceOwner := id("300"), id("301"), id("302")
	inputFields := []wire.ID{id("110"), id("111"), id("112")}
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for _, x := range []wire.ID{nameCap, limitCap, defaultCap} {
		g.Entities[x] = wire.Entity{ID: x, Schema: id("16"), Version: 1, Fields: map[wire.ID]wire.Value{}}
	}
	for x, schema := range map[wire.ID]wire.ID{i64: id("9010"), text: id("90f8"), boolean: id("9020"), configured: id("9030"), prepared: id("9030"), state: id("9030"), runtime: id("9030")} {
		g.Entities[x] = wire.Entity{ID: x, Schema: schema, Version: 1, Fields: map[wire.ID]wire.Value{}}
	}
	g.Entities[input] = record(input, "Input", inputFields, []string{"NamePrefix", "Limit", "UseDefaultLimit"}, []wire.ID{text, i64, boolean}, g.Entities)
	configParams := params(g.Entities, "config-input", []wire.ID{input})
	policyParams := params(g.Entities, "policy-input", []wire.ID{configured})
	assembleParams := params(g.Entities, "assemble", []wire.ID{configured, prepared, state})
	configResult, policyResult, assembleResult := result(g.Entities, configured), result(g.Entities, prepared), result(g.Entities, runtime)
	functions := map[string]function{
		selectKey("configuration", "Initialize"): {id: configFn, owner: configOwner, parameters: configParams, result: configResult},
		selectKey("policy", "Initialize"):        {id: policyFn, owner: policyOwner, parameters: policyParams, result: policyResult},
		selectKey("service", "Assemble"):         {id: assembleFn, owner: serviceOwner, parameters: assembleParams, result: assembleResult},
	}
	p := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for name, owner := range map[string]wire.ID{"configuration": configOwner, "policy": policyOwner, "service": serviceOwner} {
		pid := id("5" + owner.String()[31:])
		p.Entities[pid] = wire.Entity{ID: pid, Schema: id("b010"), Version: 1, Fields: map[wire.ID]wire.Value{id("b100"): {Tag: 5, Bytes: []byte(name)}}}
		p.Entities[owner] = wire.Entity{ID: owner, Schema: id("b021"), Version: 1, Fields: map[wire.ID]wire.Value{id("b210"): {Tag: 6, Reference: pid}}}
	}
	origin := id("400")
	ev := evidence{graph: g, packages: p, functions: functions, types: map[string]ownedType{
		selectKey("configuration", "Input"): {input, configOwner, origin},
		selectKey("application", "State"):   {state, id("303"), id("401")},
	}}
	in := Input{
		Fields:  []FieldSelection{{Key: "NamePrefix", OwnerPackage: "configuration", Type: TypeSelection{ID: text.String()}, Origin: TypeSelection{Package: "configuration", Name: "Input"}, Required: true, Resolution: ResolutionSelection{Kind: CapabilityValue, Capability: &nameCap}}, {Key: "Limit", OwnerPackage: "configuration", Type: TypeSelection{ID: i64.String()}, Origin: TypeSelection{Package: "configuration", Name: "Input"}, Required: true, Resolution: ResolutionSelection{Kind: CapabilityValue, Capability: &limitCap}}, {Key: "UseDefaultLimit", OwnerPackage: "configuration", Type: TypeSelection{ID: boolean.String()}, Origin: TypeSelection{Package: "configuration", Name: "Input"}, Required: true, Resolution: ResolutionSelection{Kind: CapabilityValue, Capability: &defaultCap}}},
		Runtime: []RuntimeInputSelection{{Identity: "application-state", Type: TypeSelection{Package: "application", Name: "State"}}},
		Units: []UnitSelection{
			{Key: "configuration", Callable: FunctionSelection{"configuration", "Initialize"}, Arguments: []SourceSelection{{Kind: RecordConstruction, Record: &RecordSelection{Type: TypeSelection{Package: "configuration", Name: "Input"}, Members: []MemberSelection{{"NamePrefix", SourceSelection{Kind: ResolvedField, Field: "NamePrefix"}}, {"Limit", SourceSelection{Kind: ResolvedField, Field: "Limit"}}, {"UseDefaultLimit", SourceSelection{Kind: ResolvedField, Field: "UseDefaultLimit"}}}}}}},
			{Key: "policy", Callable: FunctionSelection{"policy", "Initialize"}, Dependencies: []string{"configuration"}, Arguments: []SourceSelection{{Kind: PredecessorOK, Predecessor: "configuration"}}},
			{Key: "service", Callable: FunctionSelection{"service", "Assemble"}, Dependencies: []string{"configuration", "policy"}, Arguments: []SourceSelection{{Kind: PredecessorOK, Predecessor: "configuration"}, {Kind: PredecessorOK, Predecessor: "policy"}, {Kind: RuntimeInputSource, Runtime: "application-state"}}},
		},
	}
	got, err := resolve(in, ev)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Units) != 3 || len(got.Units[0].Arguments[0].Source.Record.Members) != 3 || got.Units[2].Arguments[2].Source.DerivedType != state {
		t.Fatalf("wrong plan: %#v", got)
	}
	bound, err := BoundModel(got)
	if err != nil || len(bound.Initializers) != 3 || bound.Initializers[2].Arguments[0].Source.Predecessor != configFn || bound.Initializers[2].Arguments[1].Source.Predecessor != policyFn {
		t.Fatalf("wrong bound conversion: %#v %v", bound, err)
	}
	base, bound2, err := Models(got)
	if err != nil || len(base.Fields) != 3 || len(base.Values) != 3 || len(base.Initializers) != 3 || len(base.Transitions) != 6 || len(bound2.Initializers) != 3 {
		t.Fatalf("wrong paired models: %#v %#v %v", base, bound2, err)
	}
	bad := in
	bad.Units = append([]UnitSelection(nil), in.Units...)
	bad.Units[2].Dependencies = []string{"policy"}
	if _, err = resolve(bad, ev); err == nil {
		t.Fatal("accepted undeclared predecessor")
	}
	bad = in
	bad.Units = append([]UnitSelection(nil), in.Units...)
	bad.Units[0].Arguments = append([]SourceSelection(nil), in.Units[0].Arguments...)
	bad.Units[0].Arguments[0].Record = &RecordSelection{Type: TypeSelection{Package: "configuration", Name: "Input"}, Members: in.Units[0].Arguments[0].Record.Members[:2]}
	if _, err = resolve(bad, ev); err == nil {
		t.Fatal("accepted partial record")
	}
	bad = in
	bad.Units = append([]UnitSelection(nil), in.Units...)
	bad.Units[2].Arguments = append([]SourceSelection(nil), in.Units[2].Arguments...)
	bad.Units[2].Arguments[2].Field = "Limit"
	if _, err = resolve(bad, ev); err == nil {
		t.Fatal("accepted multiple union arms")
	}
}

func TestResolveRejectsUnauthenticatedRunWithoutOutput(t *testing.T) {
	got, err := Resolve(t.Context(), Input{})
	if err == nil || len(got.Fields) != 0 || len(got.Units) != 0 {
		t.Fatal("accepted unauthenticated run")
	}
}

func TestPureDefaultProviderExtractsOneExactTypedExpression(t *testing.T) {
	typ, fn, lit := id("700"), id("701"), id("702")
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{typ: {ID: typ, Schema: id("9010"), Version: 1, Fields: map[wire.ID]wire.Value{id("9100"): {Tag: 3, Unsigned: 64}, id("9101"): {Tag: 2}}}, lit: {ID: lit, Schema: id("9070"), Version: 1, Fields: map[wire.ID]wire.Value{id("9700"): {Tag: 3, Unsigned: 64}, id("9701"): {Tag: 6, Reference: typ}}}, fn: {ID: fn, Schema: id("9011"), Version: 1, Fields: map[wire.ID]wire.Value{id("9111"): {Tag: 7}, id("9112"): {Tag: 6, Reference: typ}, id("9113"): {Tag: 6, Reference: lit}}}}}
	ev := evidence{graph: g, functions: map[string]function{selectKey("configuration", "DefaultLimit"): {id: fn, result: typ, body: lit}}}
	got, err := providerValue(ev, FunctionSelection{"configuration", "DefaultLimit"}, typ)
	if err != nil || got != lit {
		t.Fatalf("provider: %s %v", got, err)
	}
	bad := ev.functions[selectKey("configuration", "DefaultLimit")]
	bad.parameters = []wire.ID{id("703")}
	ev.functions[selectKey("configuration", "DefaultLimit")] = bad
	if _, err = providerValue(ev, FunctionSelection{"configuration", "DefaultLimit"}, typ); err == nil {
		t.Fatal("accepted parameterized provider")
	}
}

func record(x wire.ID, name string, fields []wire.ID, names []string, types []wire.ID, entities map[wire.ID]wire.Entity) wire.Entity {
	refs := []wire.Value{}
	for i, f := range fields {
		entities[f] = wire.Entity{ID: f, Schema: id("9031"), Version: 1, Fields: map[wire.ID]wire.Value{id("9310"): {Tag: 5, Bytes: []byte(names[i])}, id("9311"): {Tag: 6, Reference: types[i]}, id("9312"): {Tag: 3, Unsigned: uint64(i)}}}
		refs = append(refs, wire.Value{Tag: 6, Reference: f})
	}
	return wire.Entity{ID: x, Schema: id("9030"), Version: 1, Fields: map[wire.ID]wire.Value{id("9300"): {Tag: 5, Bytes: []byte(name)}, id("9301"): {Tag: 7, List: refs}}}
}
func params(entities map[wire.ID]wire.Entity, prefix string, types []wire.ID) []wire.ID {
	out := []wire.ID{}
	base := 0
	for _, b := range []byte(prefix) {
		base = base*33 + int(b)
	}
	for i, typ := range types {
		x := id(fmt.Sprintf("%x", 0x500+base+i))
		entities[x] = wire.Entity{ID: x, Schema: id("9012"), Version: 1, Fields: map[wire.ID]wire.Value{id("9121"): {Tag: 6, Reference: typ}, id("9122"): {Tag: 3, Unsigned: uint64(i)}}}
		out = append(out, x)
	}
	return out
}
func result(entities map[wire.ID]wire.Entity, ok wire.ID) wire.ID {
	x := id("6" + ok.String()[31:])
	entities[x] = wire.Entity{ID: x, Schema: id("9042"), Version: 1, Fields: map[wire.ID]wire.Value{id("9400"): {Tag: 6, Reference: ok}, id("9401"): {Tag: 6, Reference: id("1")}}}
	return x
}
