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

func TestVersionSixExtendsVersionFive(t *testing.T) {
	v5, err := Declarations(5)
	if err != nil {
		t.Fatal(err)
	}
	v6, err := Declarations(6)
	if err != nil {
		t.Fatal(err)
	}
	if len(v6) != len(v5)+1 {
		t.Fatalf("v6 schemas = %d, want %d", len(v6), len(v5)+1)
	}
	for index := range v5 {
		if v5[index].ID != v6[index].ID || v5[index].Name != v6[index].Name {
			t.Fatalf("v5 schema %d changed in v6", index)
		}
	}
}

func TestVersionSevenExtendsVersionSix(t *testing.T) {
	v6, err := Declarations(6)
	if err != nil {
		t.Fatal(err)
	}
	v7, err := Declarations(7)
	if err != nil {
		t.Fatal(err)
	}
	if len(v7) != len(v6)+1 {
		t.Fatalf("v7 schemas = %d, want %d", len(v7), len(v6)+1)
	}
	for index := range v6 {
		if v6[index].ID != v7[index].ID || v6[index].Name != v7[index].Name {
			t.Fatalf("v6 schema %d changed in v7", index)
		}
	}
}

func TestVersionEightExtendsVersionSevenWithStructuredBodies(t *testing.T) {
	v7, err := Declarations(7)
	if err != nil {
		t.Fatal(err)
	}
	v8, err := Declarations(8)
	if err != nil {
		t.Fatal(err)
	}
	if len(v8) != len(v7)+2 {
		t.Fatalf("v8 schemas = %d, want %d", len(v8), len(v7)+2)
	}
	for index := range v7 {
		if v7[index].ID != v8[index].ID || v7[index].Name != v8[index].Name {
			t.Fatalf("v7 schema %d changed in v8", index)
		}
	}
	if v8[len(v7)].Name != "Block" || v8[len(v7)+1].Name != "Return" {
		t.Fatalf("unexpected v8 schemas: %q, %q", v8[len(v7)].Name, v8[len(v7)+1].Name)
	}
}

func TestVersionNineExtendsVersionEightWithIntegerMultiply(t *testing.T) {
	v8, err := Declarations(8)
	if err != nil {
		t.Fatal(err)
	}
	v9, err := Declarations(9)
	if err != nil {
		t.Fatal(err)
	}
	if len(v9) != len(v8)+1 {
		t.Fatalf("v9 schemas = %d, want %d", len(v9), len(v8)+1)
	}
	for index := range v8 {
		if v8[index].ID != v9[index].ID || v8[index].Name != v9[index].Name {
			t.Fatalf("v8 schema %d changed in v9", index)
		}
	}
	if v9[len(v8)].Name != "IntegerMultiply" {
		t.Fatalf("unexpected v9 schema %q", v9[len(v8)].Name)
	}
}

func TestVersionTenExtendsVersionNineWithIntegerSubtract(t *testing.T) {
	v9, err := Declarations(9)
	if err != nil {
		t.Fatal(err)
	}
	v10, err := Declarations(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(v10) != len(v9)+1 {
		t.Fatalf("v10 schemas = %d, want %d", len(v10), len(v9)+1)
	}
	for index := range v9 {
		if v9[index].ID != v10[index].ID || v9[index].Name != v10[index].Name {
			t.Fatalf("v9 schema %d changed in v10", index)
		}
	}
	if v10[len(v9)].Name != "IntegerSubtract" {
		t.Fatalf("unexpected v10 schema %q", v10[len(v9)].Name)
	}
}

func TestVersionElevenExtendsVersionTenWithBooleanComposition(t *testing.T) {
	v10, err := Declarations(10)
	if err != nil {
		t.Fatal(err)
	}
	v11, err := Declarations(11)
	if err != nil {
		t.Fatal(err)
	}
	if len(v11) != len(v10)+2 {
		t.Fatalf("v11 schemas = %d, want %d", len(v11), len(v10)+2)
	}
	for index := range v10 {
		if v10[index].ID != v11[index].ID || v10[index].Name != v11[index].Name {
			t.Fatalf("v10 schema %d changed in v11", index)
		}
	}
	if v11[len(v10)].Name != "BooleanLiteral" || v11[len(v10)+1].Name != "BooleanAnd" {
		t.Fatalf("unexpected v11 schemas: %q, %q", v11[len(v10)].Name, v11[len(v10)+1].Name)
	}
}

func TestVersionTwelveFreezesGenericPureFunctionProfile(t *testing.T) {
	v11, err := Declarations(11)
	if err != nil {
		t.Fatal(err)
	}
	v12, err := Declarations(12)
	if err != nil {
		t.Fatal(err)
	}
	if len(v12) != len(v11) {
		t.Fatalf("v12 schemas = %d, want unchanged %d", len(v12), len(v11))
	}
	for index := range v11 {
		if v11[index].ID != v12[index].ID || v11[index].Name != v12[index].Name {
			t.Fatalf("v11 schema %d changed in v12", index)
		}
	}
}

func TestVersionThirteenAddsControlFlowAndText(t *testing.T) {
	v12, err := Declarations(12)
	if err != nil {
		t.Fatal(err)
	}
	v13, err := Declarations(13)
	if err != nil {
		t.Fatal(err)
	}
	if len(v13) != len(v12)+4 {
		t.Fatalf("v13 schemas = %d, want %d", len(v13), len(v12)+4)
	}
	want := []string{"If", "BooleanOr", "StringEqual", "StringConcat"}
	for index := range v12 {
		if v12[index].ID != v13[index].ID || v12[index].Name != v13[index].Name {
			t.Fatalf("v12 schema %d changed", index)
		}
	}
	for index, name := range want {
		if v13[len(v12)+index].Name != name {
			t.Fatalf("v13 schema %d = %q", index, v13[len(v12)+index].Name)
		}
	}
}

func TestVersionFourteenFreezesVariableWidthStringABIProfile(t *testing.T) {
	v13, err := Declarations(13)
	if err != nil {
		t.Fatal(err)
	}
	v14, err := Declarations(14)
	if err != nil {
		t.Fatal(err)
	}
	if len(v14) != len(v13) {
		t.Fatalf("v14 schema count = %d, want %d", len(v14), len(v13))
	}
	for index := range v13 {
		if v13[index].ID != v14[index].ID || v13[index].Name != v14[index].Name {
			t.Fatalf("v13 schema %d changed", index)
		}
	}
}

func TestVersionFifteenAddsImmutableLexicalLocals(t *testing.T) {
	v14, err := Declarations(14)
	if err != nil {
		t.Fatal(err)
	}
	v15, err := Declarations(15)
	if err != nil {
		t.Fatal(err)
	}
	if len(v15) != len(v14)+3 {
		t.Fatalf("v15 schema count = %d, want %d", len(v15), len(v14)+3)
	}
	want := []string{"LocalBinding", "BindLocal", "LocalRead"}
	for index := range v14 {
		if v14[index].ID != v15[index].ID || v14[index].Name != v15[index].Name {
			t.Fatalf("v14 schema %d changed", index)
		}
	}
	for index, name := range want {
		if v15[len(v14)+index].Name != name {
			t.Fatalf("v15 schema %d = %q", index, v15[len(v14)+index].Name)
		}
	}
}

func TestVersionSixteenFreezesMultiFunctionCallProfile(t *testing.T) {
	v15, err := Declarations(15)
	if err != nil {
		t.Fatal(err)
	}
	v16, err := Declarations(16)
	if err != nil {
		t.Fatal(err)
	}
	if len(v16) != len(v15) {
		t.Fatalf("v16 schema count = %d, want unchanged %d", len(v16), len(v15))
	}
	for index := range v15 {
		if v15[index].ID != v16[index].ID || v15[index].Name != v16[index].Name {
			t.Fatalf("v15 schema %d changed", index)
		}
	}
}

func TestVersionSeventeenFreezesCompositionalRecordProfile(t *testing.T) {
	v16, err := Declarations(16)
	if err != nil {
		t.Fatal(err)
	}
	v17, err := Declarations(17)
	if err != nil {
		t.Fatal(err)
	}
	if len(v17) != len(v16) {
		t.Fatalf("v17 schema count = %d, want %d", len(v17), len(v16))
	}
	for index := range v16 {
		if v16[index].ID != v17[index].ID || v16[index].Name != v17[index].Name {
			t.Fatalf("v16 schema %d changed", index)
		}
	}
}

func TestVersionEighteenAddsMutablePlacesAndWhile(t *testing.T) {
	v17, err := Declarations(17)
	if err != nil {
		t.Fatal(err)
	}
	v18, err := Declarations(18)
	if err != nil {
		t.Fatal(err)
	}
	if len(v18) != len(v17)+5 {
		t.Fatalf("v18 schema count = %d", len(v18))
	}
	want := []string{"MutablePlace", "DeclarePlace", "PlaceRead", "AssignPlace", "While"}
	for index, name := range want {
		if v18[len(v17)+index].Name != name {
			t.Fatalf("v18 schema = %q", v18[len(v17)+index].Name)
		}
	}
}

func TestUnsupportedVersionRejects(t *testing.T) {
	if _, err := Declarations(19); err == nil {
		t.Fatal("unsupported version accepted")
	}
}
