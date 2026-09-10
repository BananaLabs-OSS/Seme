package state

import (
	"example.test/go-uab-11/application"
	"testing"
)

func TestMigrationIsPureAndExactlyVersioned(t *testing.T) {
	old := V1{State: application.State{Name: "pilot", Values: []int64{1, 2}, Counters: map[int64]int64{1: 2}}}
	got := MigrateV1ToV2(old)
	if !got.Ok || got.Value.State.Name != "pilot" || len(got.Value.State.Values) != 2 || got.Value.State.Counters == nil || got.Value.Revision != 1 {
		t.Fatalf("migration=%#v", got)
	}
	if MigrateV1ToV2(V1{}).Ok {
		t.Fatal("invalid v1 migrated")
	}
}
