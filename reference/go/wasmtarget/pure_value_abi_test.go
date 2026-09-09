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
		{"malformed record", "wasm.pure_record_type_fields", wire.Envelope{Entities: map[wire.ID]wire.Entity{optionType: {ID: optionType, Schema: identity(0x9030), Fields: map[wire.ID]wire.Value{}}}}},
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

func TestPureValueLayoutCertifiesRecordArraySliceAndMapComposition(t *testing.T) {
	i64, record, first, second := identity(0x5101), identity(0x5102), identity(0x5103), identity(0x5104)
	array, slice, mapping := identity(0x5105), identity(0x5106), identity(0x5107)
	g := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		i64:     {ID: i64, Schema: identity(0x9010), Fields: map[wire.ID]wire.Value{identity(0x9100): unsigned(64), identity(0x9101): {Tag: 2}, identity(0x9102): unsigned(0)}},
		first:   {ID: first, Schema: identity(0x9031), Fields: map[wire.ID]wire.Value{identity(0x9310): byteValue("left"), identity(0x9311): ref(i64), identity(0x9312): unsigned(0)}},
		second:  {ID: second, Schema: identity(0x9031), Fields: map[wire.ID]wire.Value{identity(0x9310): byteValue("right"), identity(0x9311): ref(i64), identity(0x9312): unsigned(1)}},
		record:  {ID: record, Schema: identity(0x9030), Fields: map[wire.ID]wire.Value{identity(0x9301): refs(first, second)}},
		array:   {ID: array, Schema: identity(0x90f2), Fields: map[wire.ID]wire.Value{identity(0x9f20): ref(i64), identity(0x9f21): unsigned(2)}},
		slice:   {ID: slice, Schema: identity(0x90f8), Fields: map[wire.ID]wire.Value{identity(0x9f80): ref(i64)}},
		mapping: {ID: mapping, Schema: identity(0xa040), Fields: map[wire.ID]wire.Value{identity(0xa0400): ref(i64), identity(0xa0401): ref(i64)}},
	}}
	for _, test := range []struct {
		id    wire.ID
		kind  string
		fixed uint64
	}{{record, "record", 16}, {array, "array<i64,2>", 16}, {slice, "slice<i64>", 8}, {mapping, "map<i64,i64>", 8}} {
		layout, err := CertifyPureValueLayout(g, test.id)
		if err != nil {
			t.Fatalf("%s: %v", test.kind, err)
		}
		if layout.Type != test.kind || layout.FixedSize != test.fixed {
			t.Fatalf("layout = %#v", layout)
		}
	}
	sliceLayout, _ := CertifyPureValueLayout(g, slice)
	validSlice := make([]byte, 24)
	binary.LittleEndian.PutUint32(validSlice, 8)
	binary.LittleEndian.PutUint32(validSlice[4:], 2)
	if err := ValidatePureValueBytes(sliceLayout, validSlice); err != nil {
		t.Fatal(err)
	}
	mapLayout, _ := CertifyPureValueLayout(g, mapping)
	validMap := make([]byte, 40)
	binary.LittleEndian.PutUint32(validMap, 8)
	binary.LittleEndian.PutUint32(validMap[4:], 2)
	binary.LittleEndian.PutUint64(validMap[8:], uint64(1))
	binary.LittleEndian.PutUint64(validMap[24:], uint64(2))
	if err := ValidatePureValueBytes(mapLayout, validMap); err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint64(validMap[24:], uint64(1))
	if err := ValidatePureValueBytes(mapLayout, validMap); err == nil {
		t.Fatal("duplicate map key accepted")
	}

	program, function := identity(0x5110), identity(0x5111)
	parameterTypes := []wire.ID{record, array, slice, mapping}
	parameterIDs := []wire.ID{identity(0x5112), identity(0x5113), identity(0x5114), identity(0x5115)}
	g.Entities[program] = wire.Entity{ID: program, Schema: identity(0x9015), Fields: map[wire.ID]wire.Value{identity(0x9150): refs(function), identity(0x9151): ref(function)}}
	g.Entities[function] = wire.Entity{ID: function, Schema: identity(0x9011), Fields: map[wire.ID]wire.Value{identity(0x9111): refs(parameterIDs...), identity(0x9112): ref(i64)}}
	for index, id := range parameterIDs {
		g.Entities[id] = wire.Entity{ID: id, Schema: identity(0x9012), Fields: map[wire.ID]wire.Value{identity(0x9121): ref(parameterTypes[index]), identity(0x9122): unsigned(uint64(index))}}
	}
	abi, err := CertifyPureCompositeABI(g)
	if err != nil {
		t.Fatal(err)
	}
	if abi.RequestFixedSize != 48 || abi.ResponseFixedSize != 8 || len(abi.Parameters) != 4 {
		t.Fatalf("composed ABI = %#v", abi)
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
