package wasmtarget

import (
	"bytes"
	"testing"
)

func TestModuleIsDeterministicAndLayoutDerived(t *testing.T) {
	canonical := applicationLayout{requestOffsets: [3]uint64{0, 8, 16}, requestSize: 24, responseSize: 1}
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

	reordered := applicationLayout{requestOffsets: [3]uint64{0, 16, 8}, requestSize: 24, responseSize: 1}
	changed, err := module(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, changed) {
		t.Fatal("changed canonical offsets did not change codec")
	}
}

func TestModuleRejectsUnboundedLayout(t *testing.T) {
	_, err := module(applicationLayout{requestSize: 65536, responseSize: 1})
	if err == nil {
		t.Fatal("oversized record layout accepted")
	}
}
