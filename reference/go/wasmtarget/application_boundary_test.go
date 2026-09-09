package wasmtarget

import (
	"encoding/binary"
	"reflect"
	"testing"

	"seme.local/reference/canonicaleval"
)

func TestApplicationRequestUsesOneAbsoluteDescriptorSpace(t *testing.T) {
	i64 := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "i64", FixedSize: 8, Encoding: "i64"}
	text := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "text", FixedSize: 8, VariablePayload: true, MaximumPayload: 4096, Encoding: "text"}
	slice := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "slice<i64>", FixedSize: 8, VariablePayload: true, MaximumPayload: 4096, Encoding: "slice", Elements: &i64}
	boundary := PureApplicationBoundary{Parameters: []PureValueLayout{text, slice}}
	boundary.Request = aggregateApplicationRequest(boundary.Parameters)
	parameters := []canonicaleval.Value{{Kind: "text", Text: "λ"}, {Kind: "slice", Items: []canonicaleval.Value{{Kind: "i64", I64: "7"}, {Kind: "i64", I64: "-1"}}}}
	encoded, err := EncodePureApplicationRequest(boundary, parameters)
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint32(encoded[0:4]); got != uint32(boundary.Request.FixedSize) {
		t.Fatalf("text offset = %d", got)
	}
	if got := binary.LittleEndian.Uint32(encoded[8:12]); got != uint32(boundary.Request.FixedSize+2) {
		t.Fatalf("slice offset = %d", got)
	}
	decoded, err := DecodePureApplicationRequest(boundary, encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, parameters) {
		t.Fatalf("decoded = %#v", decoded)
	}

	forged := append([]byte(nil), encoded...)
	binary.LittleEndian.PutUint32(forged[8:12], uint32(boundary.Request.FixedSize)) // overlaps text payload
	if _, err = DecodePureApplicationRequest(boundary, forged); err == nil {
		t.Fatal("overlapping parameter payloads accepted")
	}
}
