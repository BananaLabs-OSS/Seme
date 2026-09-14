package executionmodule

import (
	"reflect"
	"testing"
)

func TestVersion32AddsNeutralVariantMatchingAndBytesObservation(t *testing.T) {
	previous, err := Declarations(31)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(32)
	if err != nil {
		t.Fatal(err)
	}
	want := []uint64{0xa060, 0xa061, 0xa062, 0xa063, 0xa064, 0xa065}
	if len(current) != len(previous)+len(want) {
		t.Fatalf("schema count = %d, want %d", len(current), len(previous)+len(want))
	}
	for index := range previous {
		if current[index].ID != previous[index].ID || current[index].Name != previous[index].Name {
			t.Fatalf("v31 schema %d changed", index)
		}
	}
	for index, id := range want {
		if current[len(previous)+index].ID != id {
			t.Fatalf("schema %d = %x, want %x", index, current[len(previous)+index].ID, id)
		}
	}
	resultMatch := current[len(previous)+2]
	if len(resultMatch.Fields) != 5 || resultMatch.Fields[1].Schema != 0xa060 || resultMatch.Fields[2].Schema != 0x9080 || resultMatch.Fields[3].Schema != 0xa060 || resultMatch.Fields[4].Schema != 0x9080 {
		t.Fatalf("ResultMatch fields = %#v", resultMatch.Fields)
	}
	optionMatch := current[len(previous)+3]
	if len(optionMatch.Fields) != 4 || optionMatch.Fields[1].Schema != 0x9080 || optionMatch.Fields[2].Schema != 0xa060 || optionMatch.Fields[3].Schema != 0x9080 {
		t.Fatalf("OptionMatch fields = %#v", optionMatch.Fields)
	}
}

func TestVersion31AddsNeutralOptionValues(t *testing.T) {
	previous, err := Declarations(30)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(31)
	if err != nil {
		t.Fatal(err)
	}
	want := []Schema{
		{0xa050, "OptionType", []Field{{0xa0500, "option.value_type", 5, 0, 0}}},
		{0xa051, "OptionNone", []Field{{0xa0510, "option_none.type", 5, 0xa050, 0}}},
		{0xa052, "OptionSome", []Field{{0xa0520, "option_some.type", 5, 0xa050, 0}, {0xa0521, "option_some.value", 5, 0, 0}}},
	}
	if len(current) != len(previous)+len(want) {
		t.Fatalf("schema count = %d, want %d", len(current), len(previous)+len(want))
	}
	for index := range previous {
		if current[index].ID != previous[index].ID || current[index].Name != previous[index].Name {
			t.Fatalf("v30 schema %d changed in v31", index)
		}
	}
	for index := range want {
		got := current[len(previous)+index]
		if got.ID != want[index].ID || got.Name != want[index].Name || len(got.Fields) != len(want[index].Fields) {
			t.Fatalf("v31 schema %d = %#v, want %#v", index, got, want[index])
		}
		for fieldIndex := range want[index].Fields {
			if got.Fields[fieldIndex] != want[index].Fields[fieldIndex] {
				t.Fatalf("v31 schema %d field %d = %#v, want %#v", index, fieldIndex, got.Fields[fieldIndex], want[index].Fields[fieldIndex])
			}
		}
	}
}

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

func TestVersionNineteenAddsStructuredWhen(t *testing.T) {
	previous, err := Declarations(18)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(19)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+1 || current[len(current)-1].Name != "When" {
		t.Fatalf("v19 declarations = %#v", current[len(previous):])
	}
}

func TestUnsupportedVersionRejects(t *testing.T) {
	if _, err := Declarations(59); err == nil {
		t.Fatal("unsupported version accepted")
	}
}

func TestVersion58PreservesVersion57Schema(t *testing.T) {
	previous, err := Declarations(57)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(58)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous) {
		t.Fatalf("v58 declaration count = %d, want %d", len(current), len(previous))
	}
}

func TestVersion57PreservesVersion56Schema(t *testing.T) {
	previous, err := Declarations(56)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(57)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous) {
		t.Fatalf("v57 declaration count = %d, want %d", len(current), len(previous))
	}
}

func TestVersion56PreservesVersion55Schema(t *testing.T) {
	previous, err := Declarations(55)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(56)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous) {
		t.Fatalf("v56 declaration count = %d, want %d", len(current), len(previous))
	}
}

func TestVersion55PreservesVersion54Schema(t *testing.T) {
	previous, err := Declarations(54)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(55)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous) {
		t.Fatalf("v55 declaration count = %d, want %d", len(current), len(previous))
	}
}

func TestVersion54PreservesVersion53Schema(t *testing.T) {
	previous, err := Declarations(53)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(54)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous) {
		t.Fatalf("v54 declaration count = %d, want %d", len(current), len(previous))
	}
}

func TestVersion53PreservesVersion52Schema(t *testing.T) {
	previous, err := Declarations(52)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(53)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous) {
		t.Fatalf("v53 declaration count = %d, want %d", len(current), len(previous))
	}
}

func TestVersion52PreservesVersion51Schema(t *testing.T) {
	previous, err := Declarations(51)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(52)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous) {
		t.Fatalf("v52 declaration count = %d, want %d", len(current), len(previous))
	}
}

func TestVersion51PreservesVersion50Schema(t *testing.T) {
	previous, err := Declarations(50)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(51)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous) {
		t.Fatalf("v51 declaration count = %d, want %d", len(current), len(previous))
	}
}

func TestVersion50PreservesVersion49Schema(t *testing.T) {
	previous, err := Declarations(49)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(50)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous) {
		t.Fatalf("v50 declaration count = %d, want %d", len(current), len(previous))
	}
}

func TestVersion49PreservesVersion48Schema(t *testing.T) {
	previous, err := Declarations(48)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(49)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous) {
		t.Fatalf("v49 declaration count = %d, want %d", len(current), len(previous))
	}
}

func TestVersion48PreservesVersion47Schema(t *testing.T) {
	previous, err := Declarations(47)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(48)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous) {
		t.Fatalf("v48 declaration count = %d, want %d", len(current), len(previous))
	}
}

func TestVersion47PreservesVersion46Schema(t *testing.T) {
	previous, err := Declarations(46)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(47)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous) {
		t.Fatalf("v47 declaration count = %d, want %d", len(current), len(previous))
	}
}

func TestVersion46PreservesVersion45Schema(t *testing.T) {
	previous, err := Declarations(45)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(46)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous) {
		t.Fatalf("v46 declaration count = %d, want %d", len(current), len(previous))
	}
}

func TestVersion45AddsNativeDefaultValue(t *testing.T) {
	previous, err := Declarations(44)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(45)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+1 || current[len(current)-1].Name != "NativeDefaultValue" {
		t.Fatalf("v45 declarations = %#v", current[len(previous):])
	}
}

func TestVersion44PreservesVersion43Schema(t *testing.T) {
	previous, err := Declarations(43)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(44)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous) {
		t.Fatalf("v44 declaration count = %d, want %d", len(current), len(previous))
	}
}

func TestVersion43AddsTypedNativeFieldRead(t *testing.T) {
	previous, err := Declarations(42)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(43)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+2 || current[len(previous)].ID != 0xa072 || current[len(previous)].Name != "NativeFieldRead" || current[len(previous)+1].ID != 0xa073 || current[len(previous)+1].Name != "NativeBindingRead" {
		t.Fatalf("v43 declarations = %#v", current[len(previous):])
	}
	fields := current[len(previous)].Fields
	if len(fields) != 4 || fields[0].Kind != 4 || fields[1].Kind != 4 || fields[2].Kind != 5 || fields[3].Kind != 5 {
		t.Fatalf("NativeFieldRead fields = %#v", fields)
	}
	bindingFields := current[len(previous)+1].Fields
	if len(bindingFields) != 3 || bindingFields[0].Kind != 4 || bindingFields[1].Kind != 4 || bindingFields[2].Kind != 5 {
		t.Fatalf("NativeBindingRead fields = %#v", bindingFields)
	}
}

func TestVersion42ReusesNativeTypeWithoutChangingSchema(t *testing.T) {
	previous, err := Declarations(41)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(42)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(current, previous) {
		t.Fatal("v42 provider milestone changed neutral schemas")
	}
}

func TestVersion41AddsNeutralProducts(t *testing.T) {
	previous, err := Declarations(40)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(41)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+3 || current[len(previous)].Name != "ProductType" || current[len(previous)+1].Name != "ProductProject" || current[len(previous)+2].Name != "NativeType" {
		t.Fatalf("v41 declarations = %#v", current[len(previous):])
	}
}

func TestVersion40AddsTypedNativeMethodInvocation(t *testing.T) {
	previous, err := Declarations(39)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(40)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+1 || current[len(previous)].ID != 0xa06e || current[len(previous)].Name != "NativeMethodInvocation" {
		t.Fatalf("v40 declarations = %#v", current[len(previous):])
	}
	fields := current[len(previous)].Fields
	if len(fields) != 6 || fields[3].Kind != 5 || fields[4].Card != 2 || fields[5].Kind != 5 {
		t.Fatalf("NativeMethodInvocation fields = %#v", fields)
	}
}

func TestVersion39AddsTypedNativeInvocation(t *testing.T) {
	previous, err := Declarations(38)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(39)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+1 || current[len(previous)].ID != 0xa06d || current[len(previous)].Name != "NativeInvocation" {
		t.Fatalf("v39 declarations = %#v", current[len(previous):])
	}
	fields := current[len(previous)].Fields
	if len(fields) != 5 || fields[0].Kind != 4 || fields[3].Kind != 5 || fields[3].Card != 2 || fields[4].Kind != 5 {
		t.Fatalf("NativeInvocation fields = %#v", fields)
	}
}

func TestVersion38AddsNeutralEvaluateStatement(t *testing.T) {
	previous, err := Declarations(37)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(38)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+1 || current[len(previous)].ID != 0xa06c || current[len(previous)].Name != "Evaluate" {
		t.Fatalf("v38 declarations = %#v", current[len(previous):])
	}
	fields := current[len(previous)].Fields
	if len(fields) != 1 || fields[0].ID != 0xa06c0 || fields[0].Kind != 5 {
		t.Fatalf("Evaluate fields = %#v", fields)
	}
}

func TestVersion37AddsNeutralUnit(t *testing.T) {
	previous, err := Declarations(36)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(37)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+2 || current[len(previous)].ID != 0xa06a || current[len(previous)].Name != "UnitType" || current[len(previous)+1].ID != 0xa06b || current[len(previous)+1].Name != "UnitValue" {
		t.Fatalf("v37 declarations = %#v", current[len(previous):])
	}
	fields := current[len(previous)+1].Fields
	if len(fields) != 1 || fields[0].ID != 0xa06b0 || fields[0].Kind != 5 || fields[0].Schema != 0xa06a {
		t.Fatalf("UnitValue fields = %#v", fields)
	}
}

func TestVersion36AddsBooleanNot(t *testing.T) {
	previous, err := Declarations(35)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(36)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+1 || current[len(previous)].ID != 0xa069 || current[len(previous)].Name != "BooleanNot" {
		t.Fatalf("v36 declarations = %#v", current[len(previous):])
	}
	fields := current[len(previous)].Fields
	if len(fields) != 1 || fields[0].ID != 0xa0690 || fields[0].Kind != 5 {
		t.Fatalf("BooleanNot fields = %#v", fields)
	}
}

func TestVersion35AddsOptionMapLookup(t *testing.T) {
	previous, err := Declarations(34)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(35)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+1 || current[len(previous)].ID != 0xa044 {
		t.Fatalf("v35 declarations = %#v", current[len(previous):])
	}
	fields := current[len(previous)].Fields
	if len(fields) != 3 || fields[0].ID != 0xa0440 || fields[1].ID != 0xa0441 || fields[2].ID != 0xa0442 || fields[2].Schema != 0xa050 {
		t.Fatalf("MapLookupOption fields = %#v", fields)
	}
}

func TestVersion30AddsRuntimeKeyedMaps(t *testing.T) {
	previous, err := Declarations(29)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(30)
	if err != nil {
		t.Fatal(err)
	}
	want := []uint64{0xa040, 0xa041, 0xa042, 0xa043}
	if len(current) != len(previous)+len(want) {
		t.Fatalf("schema count = %d", len(current))
	}
	for index, id := range want {
		if current[len(previous)+index].ID != id {
			t.Fatalf("schema %d = %x, want %x", index, current[len(previous)+index].ID, id)
		}
	}
}

func TestVersion29AddsMutableClosureEnvironments(t *testing.T) {
	previous, err := Declarations(28)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(29)
	if err != nil {
		t.Fatal(err)
	}
	want := []uint64{0xa030, 0xa031, 0xa032, 0xa033, 0xa034, 0xa035}
	if len(current) != len(previous)+len(want) {
		t.Fatalf("schema count = %d", len(current))
	}
	for index, id := range want {
		if current[len(previous)+index].ID != id {
			t.Fatalf("schema %d = %x, want %x", index, current[len(previous)+index].ID, id)
		}
	}
}

func TestVersion28AddsImmutableLexicalClosures(t *testing.T) {
	previous, err := Declarations(27)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(28)
	if err != nil {
		t.Fatal(err)
	}
	want := []uint64{0xa020, 0xa021, 0xa022, 0xa023, 0xa024}
	if len(current) != len(previous)+len(want) {
		t.Fatalf("schema count = %d", len(current))
	}
	for index, id := range want {
		if current[len(previous)+index].ID != id {
			t.Fatalf("schema %d = %x, want %x", index, current[len(previous)+index].ID, id)
		}
	}
}

func TestVersion27AddsInterfacesAndBoundedDynamicDispatch(t *testing.T) {
	previous, err := Declarations(26)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(27)
	if err != nil {
		t.Fatal(err)
	}
	want := []uint64{0xa010, 0xa011, 0xa012, 0xa013, 0xa014}
	if len(current) != len(previous)+len(want) {
		t.Fatalf("schema count = %d", len(current))
	}
	for index, id := range want {
		if current[len(previous)+index].ID != id {
			t.Fatalf("schema %d = %x, want %x", index, current[len(previous)+index].ID, id)
		}
	}
}

func TestVersion26AddsMethodsAndExplicitTransitions(t *testing.T) {
	previous, err := Declarations(25)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(26)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+8 {
		t.Fatalf("schema count = %d, want %d", len(current), len(previous)+8)
	}
	want := []uint64{0xa000, 0xa001, 0xa002, 0xa003, 0xa004, 0xa005, 0xa006, 0xa007}
	for index, id := range want {
		if current[len(previous)+index].ID != id {
			t.Fatalf("schema %d = %x, want %x", index, current[len(previous)+index].ID, id)
		}
	}
}

func TestVersionTwentyFiveAddsImmutableCollectionResults(t *testing.T) {
	previous, err := Declarations(24)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(25)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+2 || current[len(previous)].Name != "CollectionAppend" || current[len(previous)+1].Name != "CollectionUpdate" {
		t.Fatalf("v25 declarations = %#v", current[len(previous):])
	}
}

func TestVersionTwentyFourAddsCollectionQueries(t *testing.T) {
	previous, err := Declarations(23)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(24)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+2 || current[len(previous)].Name != "CollectionLength" || current[len(previous)+1].Name != "DynamicIndexRead" {
		t.Fatalf("v24 declarations = %#v", current[len(previous):])
	}
}

func TestVersionTwentyThreeAddsRuntimeSizedSlices(t *testing.T) {
	previous, err := Declarations(22)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(23)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+1 || current[len(previous)].Name != "SliceType" {
		t.Fatalf("v23 declarations = %#v", current[len(previous):])
	}
}

func TestVersionTwentyTwoAddsIterationBindingsAndFold(t *testing.T) {
	previous, err := Declarations(21)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(22)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+3 || current[len(previous)].Name != "IterationBinding" || current[len(previous)+1].Name != "IterationBindingRead" || current[len(previous)+2].Name != "Fold" {
		t.Fatalf("v22 declarations = %#v", current[len(previous):])
	}
}

func TestVersionTwentyOneAddsFixedArraysAndIndexRead(t *testing.T) {
	previous, err := Declarations(20)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(21)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+3 || current[len(previous)].Name != "FixedArrayType" || current[len(previous)+1].Name != "FixedArrayConstruct" || current[len(previous)+2].Name != "IndexRead" {
		t.Fatalf("v21 declarations = %#v", current[len(previous):])
	}
}

func TestVersionTwentyAddsEffectInvocation(t *testing.T) {
	previous, err := Declarations(19)
	if err != nil {
		t.Fatal(err)
	}
	current, err := Declarations(20)
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != len(previous)+1 || current[len(current)-1].Name != "EffectInvoke" {
		t.Fatalf("v20 declarations = %#v", current[len(previous):])
	}
}
