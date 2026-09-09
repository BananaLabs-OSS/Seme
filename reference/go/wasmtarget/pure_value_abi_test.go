package wasmtarget

import (
	"encoding/binary"
	"testing"

	"seme.local/reference/wire"
)

func TestPureValueLayoutsForBytesResultAndOption(t *testing.T) {
	i64, text, bytesType := identity(0x3101), identity(0x3102), identity(0x3103)
	resultType, optionType := identity(0x3104), identity(0x3105)
	graph := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		i64:        {ID: i64, Schema: identity(0x9010), Fields: map[wire.ID]wire.Value{}},
		text:       {ID: text, Schema: identity(0x9040), Fields: map[wire.ID]wire.Value{}},
		bytesType:  {ID: bytesType, Schema: identity(0x9041), Fields: map[wire.ID]wire.Value{}},
		resultType: {ID: resultType, Schema: identity(0x9042), Fields: map[wire.ID]wire.Value{identity(0x9400): ref(bytesType), identity(0x9401): ref(text)}},
		optionType: {ID: optionType, Schema: identity(0xa050), Fields: map[wire.ID]wire.Value{identity(0xa0500): ref(resultType)}},
	}}
	bytesLayout, err := CertifyPureValueLayout(graph, bytesType)
	if err != nil || bytesLayout.FixedSize != 8 || !bytesLayout.VariablePayload || bytesLayout.MaximumPayload != 4096 {
		t.Fatalf("bytes layout = %#v, %v", bytesLayout, err)
	}
	resultLayout, err := CertifyPureValueLayout(graph, resultType)
	if err != nil || resultLayout.FixedSize != 9 || len(resultLayout.Variants) != 2 || resultLayout.Variants[0].Tag != 0 || resultLayout.Variants[1].Tag != 1 {
		t.Fatalf("result layout = %#v, %v", resultLayout, err)
	}
	optionLayout, err := CertifyPureValueLayout(graph, optionType)
	if err != nil || optionLayout.FixedSize != 10 || len(optionLayout.Variants) != 2 || optionLayout.Variants[0].Payload != nil || optionLayout.Variants[1].Payload == nil {
		t.Fatalf("option layout = %#v, %v", optionLayout, err)
	}
}

func TestPureValueBytesValidateTagsDescriptorsAndInactivePayload(t *testing.T) {
	bytesLayout := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "bytes", FixedSize: 8, VariablePayload: true, MaximumPayload: 4096, Encoding: "u32le-offset-u32le-byte-length/opaque"}
	encoded := make([]byte, 11)
	binary.LittleEndian.PutUint32(encoded[0:4], 8)
	binary.LittleEndian.PutUint32(encoded[4:8], 3)
	copy(encoded[8:], []byte{0, 0xff, 1})
	if err := ValidatePureValueBytes(bytesLayout, encoded); err != nil {
		t.Fatal(err)
	}
	badDescriptor := append([]byte(nil), encoded...)
	binary.LittleEndian.PutUint32(badDescriptor[0:4], 7)
	if err := ValidatePureValueBytes(bytesLayout, badDescriptor); err == nil {
		t.Fatal("header-overlapping descriptor accepted")
	}

	boolLayout := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "bool", FixedSize: 1, Encoding: "canonical-u8-0-or-1"}
	option := PureValueLayout{Contract: "seme.pure-value-abi/v1", Type: "option<bool>", FixedSize: 2, Encoding: "u8-tag(0=none,1=some)/zeroed-absent-payload", Variants: []PureValueVariantLayout{{Tag: 0, Name: "none", Offset: 1}, {Tag: 1, Name: "some", Offset: 1, Payload: &boolLayout}}}
	for _, accepted := range [][]byte{{0, 0}, {1, 0}, {1, 1}} {
		if err := ValidatePureValueBytes(option, accepted); err != nil {
			t.Fatalf("%v: %v", accepted, err)
		}
	}
	for _, rejected := range [][]byte{{2, 0}, {0, 1}, {1, 2}} {
		if err := ValidatePureValueBytes(option, rejected); err == nil {
			t.Fatalf("accepted %v", rejected)
		}
	}
}

func TestPureValueLayoutRejectsMalformedAndRecursiveTypes(t *testing.T) {
	optionType := identity(0x3201)
	tests := []struct {
		name, want string
		graph      wire.Envelope
	}{
		{"missing", "wasm.pure_value_type_missing", wire.Envelope{Entities: map[wire.ID]wire.Entity{}}},
		{"wrong field kind", "wasm.pure_option_type_fields", wire.Envelope{Entities: map[wire.ID]wire.Entity{optionType: {ID: optionType, Schema: identity(0xa050), Fields: map[wire.ID]wire.Value{identity(0xa0500): unsigned(1)}}}}},
		{"recursive", "wasm.pure_value_type_cycle_or_size", wire.Envelope{Entities: map[wire.ID]wire.Entity{optionType: {ID: optionType, Schema: identity(0xa050), Fields: map[wire.ID]wire.Value{identity(0xa0500): ref(optionType)}}}}},
		{"unsupported", "wasm.pure_value_type_unsupported", wire.Envelope{Entities: map[wire.ID]wire.Entity{optionType: {ID: optionType, Schema: identity(0x9030), Fields: map[wire.ID]wire.Value{}}}}},
	}
	for _, item := range tests {
		t.Run(item.name, func(t *testing.T) {
			_, err := CertifyPureValueLayout(item.graph, optionType)
			if err == nil || err.Error() != item.want {
				t.Fatalf("error = %v, want %s", err, item.want)
			}
		})
	}
}

func TestPureCompositeABIDerivesCanonicalSignatureOrder(t *testing.T) {
	program, function, first, second := identity(0x3301), identity(0x3302), identity(0x3303), identity(0x3304)
	bytesType, optionType := identity(0x3305), identity(0x3306)
	graph := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		program:    {ID: program, Schema: identity(0x9015), Fields: map[wire.ID]wire.Value{identity(0x9150): refs(function), identity(0x9151): ref(function)}},
		function:   {ID: function, Schema: identity(0x9011), Fields: map[wire.ID]wire.Value{identity(0x9111): refs(first, second), identity(0x9112): ref(optionType)}},
		first:      {ID: first, Schema: identity(0x9012), Fields: map[wire.ID]wire.Value{identity(0x9121): ref(bytesType), identity(0x9122): unsigned(0)}},
		second:     {ID: second, Schema: identity(0x9012), Fields: map[wire.ID]wire.Value{identity(0x9121): ref(optionType), identity(0x9122): unsigned(1)}},
		bytesType:  {ID: bytesType, Schema: identity(0x9041), Fields: map[wire.ID]wire.Value{}},
		optionType: {ID: optionType, Schema: identity(0xa050), Fields: map[wire.ID]wire.Value{identity(0xa0500): ref(bytesType)}},
	}}
	abi, err := CertifyPureCompositeABI(graph)
	if err != nil {
		t.Fatal(err)
	}
	if abi.Contract != "seme.pure-composite-abi/v1" || abi.Provider != "seme.function-composite-v1" || abi.RequestFixedSize != 17 || abi.ResponseFixedSize != 9 || len(abi.Parameters) != 2 {
		t.Fatalf("ABI = %#v", abi)
	}
	parameter := graph.Entities[second]
	parameter.Fields[identity(0x9122)] = unsigned(0)
	graph.Entities[second] = parameter
	if _, err := CertifyPureCompositeABI(graph); err == nil {
		t.Fatal("duplicate parameter position accepted")
	}
}
