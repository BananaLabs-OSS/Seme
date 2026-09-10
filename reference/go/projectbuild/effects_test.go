package projectbuild

import (
	"strings"
	"testing"

	"seme.local/reference/goprovider"
	"seme.local/reference/wire"
)

func TestDeriveEffectsUsesDirectCallableOwnership(t *testing.T) {
	functionSchema := mustID("00000000000000000000000000009011")
	invokeSchema := mustID("000000000000000000000000000090f1")
	effectSchema := mustID("00000000000000000000000000000015")
	caller, callee := effectTestID(1), effectTestID(2)
	call, invoke, effect := effectTestID(3), effectTestID(4), effectTestID(5)
	program := effectTestID(6)
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		program: {ID: program, Schema: mustID("00000000000000000000000000009015"), Fields: map[wire.ID]wire.Value{effectTestID(19): {Tag: 6, Reference: caller}}},
		caller:  {ID: caller, Schema: functionSchema, Fields: map[wire.ID]wire.Value{effectTestID(20): {Tag: 6, Reference: call}}},
		call:    {ID: call, Schema: effectTestID(21), Fields: map[wire.ID]wire.Value{effectTestID(22): {Tag: 6, Reference: callee}}},
		callee:  {ID: callee, Schema: functionSchema, Fields: map[wire.ID]wire.Value{effectTestID(23): {Tag: 6, Reference: invoke}}},
		invoke:  {ID: invoke, Schema: invokeSchema, Fields: map[wire.ID]wire.Value{mustID("00000000000000000000000000009f10"): {Tag: 6, Reference: effect}}},
		effect:  {ID: effect, Schema: effectSchema},
	}}
	packages := []goprovider.PackageMetadata{
		{Name: "example/a", Members: []goprovider.PackageFunctionMetadata{{ID: caller.String()}}},
		{Name: "example/b", Members: []goprovider.PackageFunctionMetadata{{ID: callee.String()}}},
	}
	got, err := deriveEffects(e, packages)
	if err != nil {
		t.Fatal(err)
	}
	if len(got["example/a"]) != 0 || len(got["example/b"]) != 1 || got["example/b"][0] != effect {
		t.Fatalf("wrong direct ownership: %#v", got)
	}
}

func TestDeriveEffectsSupportsReceiverCallablesAndSharedRequirements(t *testing.T) {
	methodSchema := mustID("0000000000000000000000000000a002")
	invokeSchema := mustID("000000000000000000000000000090f1")
	effectSchema := mustID("00000000000000000000000000000015")
	method, invoke, effect := effectTestID(31), effectTestID(32), effectTestID(33)
	program := effectTestID(30)
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		program: {ID: program, Schema: mustID("00000000000000000000000000009015"), Fields: map[wire.ID]wire.Value{effectTestID(29): {Tag: 6, Reference: method}}},
		method:  {ID: method, Schema: methodSchema, Fields: map[wire.ID]wire.Value{effectTestID(34): {Tag: 6, Reference: invoke}}},
		invoke:  {ID: invoke, Schema: invokeSchema, Fields: map[wire.ID]wire.Value{mustID("00000000000000000000000000009f10"): {Tag: 6, Reference: effect}}},
		effect:  {ID: effect, Schema: effectSchema},
	}}
	packages := []goprovider.PackageMetadata{{Name: "example/model", Supplemental: []goprovider.SemanticDeclarationMetadata{{Declaration: method.String(), Kind: goprovider.SemanticMethod}}}}
	got, err := deriveEffects(e, packages)
	if err != nil || len(got["example/model"]) != 1 || got["example/model"][0] != effect {
		t.Fatalf("method effects=%#v err=%v", got, err)
	}

	other := effectTestID(35)
	e.Entities[other] = wire.Entity{ID: other, Schema: methodSchema, Fields: map[wire.ID]wire.Value{effectTestID(36): {Tag: 6, Reference: invoke}}}
	root := e.Entities[program]
	root.Fields[effectTestID(29)] = wire.Value{Tag: 7, List: []wire.Value{{Tag: 6, Reference: method}, {Tag: 6, Reference: other}}}
	e.Entities[program] = root
	packages = append(packages, goprovider.PackageMetadata{Name: "example/other", Supplemental: []goprovider.SemanticDeclarationMetadata{{Declaration: other.String(), Kind: goprovider.SemanticMethod}}})
	got, err = deriveEffects(e, packages)
	if err != nil || len(got["example/model"]) != 1 || len(got["example/other"]) != 1 || got["example/model"][0] != effect || got["example/other"][0] != effect {
		t.Fatalf("shared effect requirements=%#v err=%v", got, err)
	}
}

func TestDeriveEffectsRejectsUnownedCanonicalEffect(t *testing.T) {
	effect := effectTestID(40)
	program := effectTestID(41)
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		program: {ID: program, Schema: mustID("00000000000000000000000000009015"), Fields: map[wire.ID]wire.Value{effectTestID(42): {Tag: 6, Reference: effect}}},
		effect:  {ID: effect, Schema: mustID("00000000000000000000000000000015")},
	}}
	if _, err := deriveEffects(e, nil); err == nil || !strings.Contains(err.Error(), "effect_unowned") {
		t.Fatalf("unowned effect accepted: %v", err)
	}
}

func effectTestID(last byte) wire.ID {
	var out wire.ID
	out[0] = 0x80
	out[len(out)-1] = last
	return out
}
