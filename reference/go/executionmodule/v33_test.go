package executionmodule

import "testing"

func TestVersion33AddsNeutralImmutableRemoval(t *testing.T) {
	schemas, err := Declarations(33)
	if err != nil {
		t.Fatal(err)
	}
	want := map[uint64]bool{0xa066: false, 0xa067: false}
	for _, schema := range schemas {
		if _, ok := want[schema.ID]; ok {
			want[schema.ID] = true
		}
	}
	for id, found := range want {
		if !found {
			t.Fatalf("missing schema %x", id)
		}
	}
}
