package wasmtarget

import (
	"bytes"
	"testing"

	"seme.local/reference/wire"
)

func TestModuleIsDeterministicAndLayoutDerived(t *testing.T) {
	canonical := applicationLayout{helperInstructions: []byte{0x20, 0, 0x20, 1, 0x7c, 0x20, 2, 0x57}, requestOffsets: [3]uint64{0, 8, 16}, requestHeaderSize: 32, responseBoolOffset: 1, responseHeaderSize: 10, errorMessage: []byte("subject required")}
	first, err := module(canonical)
	if err != nil {
		t.Fatal(err)
	}
	second, err := module(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("equal layouts produced different modules")
	}

	reordered := applicationLayout{helperInstructions: canonical.helperInstructions, requestOffsets: [3]uint64{0, 16, 8}, requestHeaderSize: 32, responseBoolOffset: 1, responseHeaderSize: 10, errorMessage: []byte("subject required")}
	changed, err := module(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, changed) {
		t.Fatal("changed canonical offsets did not change codec")
	}

	helperReordered := canonical
	helperReordered.helperInstructions = []byte{0x20, 1, 0x20, 0, 0x7c, 0x20, 2, 0x57}
	changed, err = module(helperReordered)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, changed) {
		t.Fatal("changed canonical helper operands did not change code")
	}
}

func TestLowerHelperIntegerComposesMultiplyAndAdd(t *testing.T) {
	typeID := identity(0x100)
	parameterID := identity(0x101)
	readID := identity(0x102)
	literalID := identity(0x103)
	addID := identity(0x104)
	multiplyID := identity(0x105)
	graph := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		typeID: {ID: typeID, Schema: identity(0x9010), Fields: map[wire.ID]wire.Value{
			identity(0x9100): {Tag: 3, Unsigned: 64}, identity(0x9101): {Tag: 2}, identity(0x9102): {Tag: 3, Unsigned: 0},
		}},
		parameterID: {ID: parameterID, Schema: identity(0x9012)},
		readID: {ID: readID, Schema: identity(0x9013), Fields: map[wire.ID]wire.Value{
			identity(0x9130): {Tag: 6, Reference: parameterID},
		}},
		literalID: {ID: literalID, Schema: identity(0x9070), Fields: map[wire.ID]wire.Value{
			identity(0x9700): {Tag: 3, Unsigned: 3}, identity(0x9701): {Tag: 6, Reference: typeID},
		}},
		addID: {ID: addID, Schema: identity(0x9014), Fields: map[wire.ID]wire.Value{
			identity(0x9140): {Tag: 6, Reference: readID}, identity(0x9141): {Tag: 6, Reference: literalID}, identity(0x9142): {Tag: 6, Reference: typeID},
		}},
		multiplyID: {ID: multiplyID, Schema: identity(0x9090), Fields: map[wire.ID]wire.Value{
			identity(0x9900): {Tag: 6, Reference: addID}, identity(0x9901): {Tag: 6, Reference: literalID}, identity(0x9902): {Tag: 6, Reference: typeID},
		}},
	}}
	budget := 16
	got, err := lowerHelperInteger(graph, multiplyID, map[wire.ID]byte{parameterID: 2}, map[byte]bool{}, map[wire.ID]bool{}, &budget)
	if err != nil {
		t.Fatal(err)
	}
	// local.get 2; i64.const 3; i64.add; i64.const 3; i64.mul
	want := []byte{0x20, 0x02, 0x42, 0x03, 0x7c, 0x42, 0x03, 0x7e}
	if !bytes.Equal(got, want) {
		t.Fatalf("instructions = %x, want %x", got, want)
	}
}

func TestLowerHelperIntegerPreservesSubtractionOrder(t *testing.T) {
	typeID := identity(0x200)
	leftParameterID := identity(0x201)
	rightParameterID := identity(0x202)
	leftReadID := identity(0x203)
	rightReadID := identity(0x204)
	subtractID := identity(0x205)
	graph := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		typeID: {ID: typeID, Schema: identity(0x9010), Fields: map[wire.ID]wire.Value{
			identity(0x9100): {Tag: 3, Unsigned: 64}, identity(0x9101): {Tag: 2}, identity(0x9102): {Tag: 3, Unsigned: 0},
		}},
		leftParameterID:  {ID: leftParameterID, Schema: identity(0x9012)},
		rightParameterID: {ID: rightParameterID, Schema: identity(0x9012)},
		leftReadID: {ID: leftReadID, Schema: identity(0x9013), Fields: map[wire.ID]wire.Value{
			identity(0x9130): {Tag: 6, Reference: leftParameterID},
		}},
		rightReadID: {ID: rightReadID, Schema: identity(0x9013), Fields: map[wire.ID]wire.Value{
			identity(0x9130): {Tag: 6, Reference: rightParameterID},
		}},
		subtractID: {ID: subtractID, Schema: identity(0x90a0), Fields: map[wire.ID]wire.Value{
			identity(0x9a00): {Tag: 6, Reference: leftReadID}, identity(0x9a01): {Tag: 6, Reference: rightReadID}, identity(0x9a02): {Tag: 6, Reference: typeID},
		}},
	}}
	budget := 8
	got, err := lowerHelperInteger(graph, subtractID, map[wire.ID]byte{leftParameterID: 4, rightParameterID: 7}, map[byte]bool{}, map[wire.ID]bool{}, &budget)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x20, 0x04, 0x20, 0x07, 0x7d}
	if !bytes.Equal(got, want) {
		t.Fatalf("instructions = %x, want ordered %x", got, want)
	}
}

func TestModuleRejectsUnboundedLayout(t *testing.T) {
	_, err := module(applicationLayout{helperInstructions: []byte{0x41, 0}, requestHeaderSize: 65536, responseHeaderSize: 10, errorMessage: []byte("subject required")})
	if err == nil {
		t.Fatal("oversized record layout accepted")
	}
	_, err = module(applicationLayout{requestHeaderSize: 32, responseHeaderSize: 10, errorMessage: []byte("subject required")})
	if err == nil {
		t.Fatal("empty helper instructions accepted")
	}
}
