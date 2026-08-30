package wasmtarget

import (
	"bytes"
	"testing"
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
