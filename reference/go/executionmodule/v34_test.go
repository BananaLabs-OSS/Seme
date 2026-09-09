package executionmodule

import "testing"

func TestV34AddsBoundedSliceConstruction(t *testing.T) {
	schemas, err := Declarations(34)
	if err != nil {
		t.Fatal(err)
	}
	for _, schema := range schemas {
		if schema.ID == 0xa068 {
			if len(schema.Fields) != 2 || schema.Fields[0].ID != 0xa0680 || schema.Fields[1].ID != 0xa0681 || schema.Fields[1].Card != 2 {
				t.Fatalf("SliceConstruct=%#v", schema)
			}
			return
		}
	}
	t.Fatal("SliceConstruct missing")
}

func TestV34SliceConstructionBounds(t *testing.T) {
	for _, count := range []int{0, 512} {
		if err := ValidateSliceConstructElementCount(count); err != nil {
			t.Fatalf("count %d: %v", count, err)
		}
	}
	if ValidateSliceConstructElementCount(513) == nil {
		t.Fatal("513 elements accepted")
	}
}
