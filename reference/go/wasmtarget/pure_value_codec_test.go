package wasmtarget

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	"seme.local/reference/canonicaleval"
)

func TestPureValueCertifiedWholeMessageCapacity(t *testing.T) {
	layout := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "bytes", FixedSize: 8, VariablePayload: true, MaximumPayload: PureValueMaximumMessageSize - 8, Encoding: "bytes"}
	if got, err := MaximumPureValueEncodedSize(layout); err != nil || got != PureValueMaximumMessageSize {
		t.Fatalf("maximum = %d, %v", got, err)
	}
	exact := make([]byte, PureValueMaximumMessageSize)
	binary.LittleEndian.PutUint32(exact[0:4], 8)
	binary.LittleEndian.PutUint32(exact[4:8], uint32(PureValueMaximumMessageSize-8))
	if err := ValidatePureValueBytes(layout, exact); err != nil {
		t.Fatalf("exact maximum rejected: %v", err)
	}
	if err := ValidatePureValueBytes(layout, append(exact, 0)); err == nil {
		t.Fatal("maximum plus one accepted")
	}
	over := layout
	over.MaximumPayload++
	if _, err := MaximumPureValueEncodedSize(over); err == nil {
		t.Fatal("over-ceiling layout accepted")
	}
	over.FixedSize = ^uint64(0)
	if _, err := MaximumPureValueEncodedSize(over); err == nil {
		t.Fatal("overflowing layout accepted")
	}
}

func TestPureValueMultipleVariableRangesAreExactlyContiguous(t *testing.T) {
	child := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "bytes", FixedSize: 8, VariablePayload: true, MaximumPayload: 16, Encoding: "bytes"}
	layout := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "record", FixedSize: 16, VariablePayload: true, MaximumPayload: 32, Encoding: "record", Fields: []PureValueFieldLayout{{Name: "A", Offset: 0, Value: child}, {Name: "B", Offset: 8, Value: child}}}
	value := canonicaleval.Value{Kind: "record", Fields: map[string]canonicaleval.Value{"A": {Kind: "bytes", Bytes: "010203"}, "B": {Kind: "bytes", Bytes: "0405"}}}
	encoded, err := EncodePureValue(layout, value)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePureValueBytes(layout, encoded); err != nil {
		t.Fatal(err)
	}
	gap := append([]byte(nil), encoded...)
	binary.LittleEndian.PutUint32(gap[8:12], binary.LittleEndian.Uint32(gap[8:12])+1)
	if err := ValidatePureValueBytes(layout, gap); err == nil {
		t.Fatal("payload gap accepted")
	}
	overlap := append([]byte(nil), encoded...)
	binary.LittleEndian.PutUint32(overlap[8:12], binary.LittleEndian.Uint32(overlap[0:4]))
	if err := ValidatePureValueBytes(layout, overlap); err == nil {
		t.Fatal("overlapping payload ranges accepted")
	}
}

func TestEncodePureValueCumulativeStateResult(t *testing.T) {
	i64 := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "i64", FixedSize: 8, Encoding: "little-endian-twos-complement-i64-modular"}
	text := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "text", FixedSize: 8, VariablePayload: true, MaximumPayload: 4096, Encoding: "u32le-offset-u32le-byte-length/utf8-scalar-exact"}
	slice := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "slice<i64>", FixedSize: 8, VariablePayload: true, MaximumPayload: 4096, Encoding: "u32le-offset-u32le-count/packed-elements", Elements: &i64}
	mapping := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "map<i64,i64>", FixedSize: 8, VariablePayload: true, MaximumPayload: 4096, Encoding: "u32le-offset-u32le-count/sorted-unique-entries", Key: &i64, Value: &i64}
	state := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "record", FixedSize: 24, VariablePayload: true, MaximumPayload: 12288, Encoding: "ordered-inline-fields", Fields: []PureValueFieldLayout{{"Name", 0, text}, {"Values", 8, slice}, {"Counters", 16, mapping}}}
	transition := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "transition<record,i64>", FixedSize: 32, VariablePayload: true, MaximumPayload: 12288, Encoding: "state-then-result/ordered-inline-fields", Fields: []PureValueFieldLayout{{"state", 0, state}, {"result", 24, i64}}}
	result := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "result<transition<record,i64>,i64>", FixedSize: 33, VariablePayload: true, MaximumPayload: 12288, Encoding: "u8-tag(0=ok,1=error)/zeroed-inactive-union-payload", Variants: []PureValueVariantLayout{{0, "ok", 1, &transition}, {1, "error", 1, &i64}}}
	stateValue := canonicaleval.Value{Kind: "record", Fields: map[string]canonicaleval.Value{
		"Name":   {Kind: "text", Text: "雪"},
		"Values": {Kind: "slice", Items: []canonicaleval.Value{{Kind: "i64", I64: "4"}, {Kind: "i64", I64: "9"}}},
		"Counters": {Kind: "map", ValueType: "i64", Entries: []canonicaleval.Entry{
			{Key: canonicaleval.Value{Kind: "i64", I64: "-2"}, Value: canonicaleval.Value{Kind: "i64", I64: "7"}},
			{Key: canonicaleval.Value{Kind: "i64", I64: "5"}, Value: canonicaleval.Value{Kind: "i64", I64: "11"}},
		}},
	}}
	transitionValue := canonicaleval.Value{Kind: "transition", State: &stateValue, Result: &canonicaleval.Value{Kind: "i64", I64: "18"}}
	value := canonicaleval.Value{Kind: "result", Variant: "ok", Payload: &transitionValue}
	first, err := EncodePureValue(result, value)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EncodePureValue(result, value)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("encoding is nondeterministic")
	}
	if len(first) != 84 {
		t.Fatalf("encoded size %d, want 84", len(first))
	}
	decoded, err := DecodePureValue(result, first)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, value) {
		t.Fatalf("round trip mismatch\n got %#v\nwant %#v", decoded, value)
	}
}

func TestEncodePureValueRejectsMalformedAndUnsortedValues(t *testing.T) {
	i64 := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "i64", FixedSize: 8, Encoding: "little-endian-twos-complement-i64-modular"}
	mapping := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "map<i64,i64>", FixedSize: 8, VariablePayload: true, MaximumPayload: 4096, Encoding: "u32le-offset-u32le-count/sorted-unique-entries", Key: &i64, Value: &i64}
	bad := canonicaleval.Value{Kind: "map", Entries: []canonicaleval.Entry{{Key: canonicaleval.Value{Kind: "i64", I64: "2"}, Value: canonicaleval.Value{Kind: "i64", I64: "0"}}, {Key: canonicaleval.Value{Kind: "i64", I64: "1"}, Value: canonicaleval.Value{Kind: "i64", I64: "0"}}}}
	if _, err := EncodePureValue(mapping, bad); err == nil {
		t.Fatal("unsorted map accepted")
	}
	if _, err := EncodePureValue(i64, canonicaleval.Value{Kind: "i64", I64: "9223372036854775808"}); err == nil {
		t.Fatal("out-of-range i64 accepted")
	}
}
