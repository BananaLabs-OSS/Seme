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

func TestVersionFourExtendsVersionThree(t *testing.T) {
	v3, err := Declarations(3)
	if err != nil {
		t.Fatal(err)
	}
	v4, err := Declarations(4)
	if err != nil {
		t.Fatal(err)
	}
	if len(v4) != len(v3)+5 {
		t.Fatalf("v4 schemas = %d, want %d", len(v4), len(v3)+5)
	}
	for index := range v3 {
		if v3[index].ID != v4[index].ID || v3[index].Name != v4[index].Name {
			t.Fatalf("v3 schema %d changed in v4", index)
		}
	}
}

func TestVersionFiveExtendsVersionFour(t *testing.T) {
	v4, err := Declarations(4)
	if err != nil {
		t.Fatal(err)
	}
	v5, err := Declarations(5)
	if err != nil {
		t.Fatal(err)
	}
	if len(v5) != len(v4)+3 {
		t.Fatalf("v5 schemas = %d, want %d", len(v5), len(v4)+3)
	}
	for index := range v4 {
		if v4[index].ID != v5[index].ID || v4[index].Name != v5[index].Name {
			t.Fatalf("v4 schema %d changed in v5", index)
		}
	}
}

func TestUnsupportedVersionRejects(t *testing.T) {
	if _, err := Declarations(6); err == nil {
		t.Fatal("unsupported version accepted")
	}
}
