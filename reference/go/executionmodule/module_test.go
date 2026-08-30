package executionmodule

import "testing"

func TestVersionThreeExtendsVersionTwo(t *testing.T) {
	v2, err := Declarations(2)
	if err != nil {
		t.Fatal(err)
	}
	v3, err := Declarations(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(v3) != len(v2)+4 {
		t.Fatalf("v3 schemas = %d, want %d", len(v3), len(v2)+4)
	}
	for index := range v2 {
		if v2[index].ID != v3[index].ID || v2[index].Name != v3[index].Name {
			t.Fatalf("v2 schema %d changed in v3", index)
		}
	}
}

func TestUnsupportedVersionRejects(t *testing.T) {
	if _, err := Declarations(4); err == nil {
		t.Fatal("unsupported version accepted")
	}
}
