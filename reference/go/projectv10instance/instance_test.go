package projectv10instance

import (
	"seme.local/reference/durableinstance"
	"seme.local/reference/wire"
	"testing"
)

func TestOneRejectsMissingAndDuplicateRoots(t *testing.T) {
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	if _, err := one(e, id("e024")); err == nil {
		t.Fatal("missing accepted")
	}
	a, b := id("1"), id("2")
	e.Entities[a] = wire.Entity{ID: a, Schema: id("e024")}
	e.Entities[b] = wire.Entity{ID: b, Schema: id("e024")}
	if _, err := one(e, id("e024")); err == nil {
		t.Fatal("duplicate accepted")
	}
}
func TestStableIdentityBindsBothInputs(t *testing.T) {
	a := stable(Inputs{ProjectV9: []byte("p"), Durable: durable([]byte("d"))}, "x")
	b := stable(Inputs{ProjectV9: []byte("p"), Durable: durable([]byte("e"))}, "x")
	if a == b {
		t.Fatal("durable artifact not bound")
	}
}
func durable(b []byte) (d durableinstance.Inputs) { d.Artifact = b; return d }
