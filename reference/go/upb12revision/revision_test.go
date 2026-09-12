package upb12revision

import (
	"seme.local/reference/wire"
	"testing"
)

func TestVerifyAcceptsOnlyOneIdentityBoundTransition(t *testing.T) {
	prior, result, target, field := pair(t)
	if err := Verify(prior, result, target, field, []byte("InitializePolicy"), []byte("BuildPolicy")); err != nil {
		t.Fatal(err)
	}
	graph, _ := wire.Decode(result)
	entity := graph.Entities[target]
	entity.Version = 2
	graph.Entities[target] = entity
	changed, _ := wire.Encode(graph)
	if err := Verify(prior, changed, target, field, []byte("InitializePolicy"), []byte("BuildPolicy")); err == nil {
		t.Fatal("accepted additional semantic change")
	}
}

func pair(t *testing.T) ([]byte, []byte, wire.ID, wire.ID) {
	t.Helper()
	target, field := id(0x2001), id(0x9110)
	base := wire.Envelope{Module: id(0x9000), Revision: id(1), Entities: map[wire.ID]wire.Entity{target: {ID: target, Schema: id(0x9011), Version: 1, Fields: map[wire.ID]wire.Value{field: {Tag: 5, Bytes: []byte("InitializePolicy")}}}}}
	prior, err := wire.Encode(base)
	if err != nil {
		t.Fatal(err)
	}
	entity := base.Entities[target]
	entity.Fields[field] = wire.Value{Tag: 5, Bytes: []byte("BuildPolicy")}
	base.Entities[target] = entity
	base.Revision = id(2)
	result, err := wire.Encode(base)
	if err != nil {
		t.Fatal(err)
	}
	return prior, result, target, field
}
func id(value uint64) (out wire.ID) {
	for index := 15; index >= 0 && value > 0; index-- {
		out[index] = byte(value)
		value >>= 8
	}
	return
}
