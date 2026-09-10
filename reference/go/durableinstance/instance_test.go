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
