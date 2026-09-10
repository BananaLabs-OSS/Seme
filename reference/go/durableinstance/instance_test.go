package durableinstance

import (
	"seme.local/reference/wire"
	"testing"
)

func TestNeutralIdentityAndTypeUnion(t *testing.T) {
	for _, s := range []string{"state.counter-v1", "app/state_2"} {
		if !name(s) {
			t.Fatalf("rejected %s", s)
		}
	}
	for _, s := range []string{"", "State", "a b", "../x"} {
		if name(s) {
			t.Fatalf("accepted %s", s)
		}
	}
	for _, s := range []string{"9010", "9020", "9040", "9041", "9042", "90f2", "90f8", "a010", "a020", "a040", "a050"} {
		if !typeEntity(wire.Entity{Schema: id(s)}) {
			t.Fatalf("type rejected %s", s)
		}
	}
	if typeEntity(wire.Entity{Schema: id("9011")}) {
		t.Fatal("function accepted as type")
	}
}
func TestBoundsAndStableIdentity(t *testing.T) {
	a := stable([]byte("project"), "family", "x")
	b := stable([]byte("project"), "family", "x")
	c := stable([]byte("project"), "family", "y")
	if a != b || a == c {
		t.Fatal("stable identity")
	}
	if MaxPayloadBytes != 1<<20 {
		t.Fatal("bound changed")
	}
}

func TestPureResultFunctionChecksExactArgumentsAndEffects(t *testing.T) {
	fn, p, result, body := id("100"), id("101"), id("102"), id("103")
	arg, okType, errType := id("200"), id("201"), id("202")
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		fn:     entity(fn, "9011", map[string]wire.Value{"9110": blob("validate"), "9111": list(p), "9112": ref(result), "9113": ref(body)}),
		p:      entity(p, "9012", map[string]wire.Value{"9120": blob("state"), "9121": ref(arg), "9122": u(0)}),
		result: entity(result, "9042", map[string]wire.Value{"9400": ref(okType), "9401": ref(errType)}),
		body:   entity(body, "913", map[string]wire.Value{"9130": ref(p)}),
		arg:    {ID: arg, Schema: id("9040"), Version: 1, Fields: map[wire.ID]wire.Value{}},
	}}
	if !pureResultFunction(e, fn, arg, okType, errType, true) {
		t.Fatal("valid typed result rejected")
	}
	if pureResultFunction(e, fn, arg, arg, errType, true) || pureResultFunction(e, fn, arg, okType, arg, true) || pureResultFunction(e, fn, arg, okType, errType, false) {
		t.Fatal("wrong type/owner accepted")
	}
	q := e.Entities[body]
	q.Schema = id("90f1")
	e.Entities[body] = q
	if pureResultFunction(e, fn, arg, okType, errType, true) {
		t.Fatal("effectful function accepted")
	}
}

func TestOperationIdentitiesAndNarrowBounds(t *testing.T) {
	if LoadEffectIdentity == CompareExchangeEffectIdentity || CodecIdentity != "seme.durable-state.canonical.v1" || MaxKeyBytes != 512 || MaxPayloadBytes != 1<<20 {
		t.Fatal("profile drift")
	}
}
