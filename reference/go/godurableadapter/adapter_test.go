package godurableadapter

import (
	"testing"

	"seme.local/reference/wire"
)

func TestSignedIntegerTypeUsesWireBooleanTag(t *testing.T) {
	typeID := id("a001")
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		typeID: {ID: typeID, Schema: id("9010"), Version: 1, Fields: map[wire.ID]wire.Value{
			id("9100"): {Tag: 3, Unsigned: 64},
			id("9101"): {Tag: 2},
			id("9102"): {Tag: 3},
		}},
	}}
	got, err := oneType(e, "9010", func(q wire.Entity) bool {
		return q.Fields[id("9100")].Tag == 3 && q.Fields[id("9100")].Unsigned == 64 && q.Fields[id("9101")].Tag == 2
	})
	if err != nil || got != typeID {
		t.Fatalf("got=%s err=%v", got, err)
	}
}
