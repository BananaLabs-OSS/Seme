package wasmtarget

import (
	"strings"
	"testing"

	"seme.local/reference/wire"
)

func TestPureCertificateControlsLowering(t *testing.T) {
	graph := certifiedTestGraph()
	certificate, err := CertifyPureFunction(graph)
	if err != nil {
		t.Fatal(err)
	}
	first, firstABI, err := LowerCertifiedPureFunction(certificate)
	if err != nil {
		t.Fatal(err)
	}
	// Certification owns the executable plan; later mutation of the caller's
	// graph cannot alter the certified artifact.
	delete(graph.Entities, identity(0x2007))
	second, secondABI, err := LowerCertifiedPureFunction(certificate)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) || firstABI.RequestSize != 16 || firstABI.ResponseSize != 8 || secondABI.Result.Type != "i64" {
		t.Fatalf("unstable certificate: first=%#v second=%#v", firstABI, secondABI)
	}
	if _, _, err := LowerCertifiedPureFunction(PureFunctionCertificate{}); err == nil {
		t.Fatal("zero certificate lowered")
	}
}

func TestPureCertificateRejectsMalformedPrograms(t *testing.T) {
	tests := []struct {
		name string
		want string
		edit func(map[wire.ID]wire.Entity)
	}{
		{"entry outside function membership", "wasm.pure_entry_membership", func(entities map[wire.ID]wire.Entity) {
			program := entities[identity(0x2000)]
			program.Fields[identity(0x9150)] = refs(identity(0x2002))
			entities[program.ID] = program
		}},
		{"malformed function list", "wasm.pure_program_functions", func(entities map[wire.ID]wire.Entity) {
			program := entities[identity(0x2000)]
			program.Fields[identity(0x9150)] = ref(identity(0x2001))
			entities[program.ID] = program
		}},
		{"duplicate parameter", "wasm.pure_parameter_membership", func(entities map[wire.ID]wire.Entity) {
			function := entities[identity(0x2001)]
			function.Fields[identity(0x9111)] = refs(identity(0x2004), identity(0x2004))
			entities[function.ID] = function
		}},
		{"parameter order", "wasm.pure_parameter_order", func(entities map[wire.ID]wire.Entity) {
			parameter := entities[identity(0x2004)]
			parameter.Fields[identity(0x9122)] = unsigned(1)
			entities[parameter.ID] = parameter
		}},
		{"illegal block statement", "wasm.pure_return", func(entities map[wire.ID]wire.Entity) {
			block := entities[identity(0x2002)]
			block.Fields[identity(0x9800)] = refs(identity(0x2007))
			entities[block.ID] = block
		}},
		{"return result mismatch", "wasm.pure_integer_add_type", func(entities map[wire.ID]wire.Entity) {
			function := entities[identity(0x2001)]
			function.Fields[identity(0x9112)] = ref(identity(0x2010))
			entities[function.ID] = function
		}},
		{"impure call", "wasm.pure_unsupported_expression", func(entities map[wire.ID]wire.Entity) {
			callID := identity(0x2011)
			entities[callID] = wire.Entity{ID: callID, Schema: identity(0x9060), Fields: map[wire.ID]wire.Value{}}
			returned := entities[identity(0x2003)]
			returned.Fields[identity(0x9810)] = refs(callID)
			entities[returned.ID] = returned
		}},
		{"expression cycle", "wasm.pure_expression_cycle_or_size", func(entities map[wire.ID]wire.Entity) {
			add := entities[identity(0x2007)]
			add.Fields[identity(0x9140)] = ref(add.ID)
			entities[add.ID] = add
		}},
		{"malformed return list", "wasm.pure_return_values", func(entities map[wire.ID]wire.Entity) {
			returned := entities[identity(0x2003)]
			returned.Fields[identity(0x9810)] = ref(identity(0x2007))
			entities[returned.ID] = returned
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			graph := certifiedTestGraph()
			test.edit(graph.Entities)
			if _, err := CertifyPureFunction(graph); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestPureCertificateRejectsExpressionOverBudget(t *testing.T) {
	graph := certifiedTestGraph()
	root := identity(0x2006)
	for index := uint64(0); index < 4097; index++ {
		id := identity(0x3000 + index)
		graph.Entities[id] = wire.Entity{ID: id, Schema: identity(0x9014), Fields: map[wire.ID]wire.Value{
			identity(0x9140): ref(root), identity(0x9141): ref(identity(0x2006)), identity(0x9142): ref(identity(0x2008)),
		}}
		root = id
	}
	returned := graph.Entities[identity(0x2003)]
	returned.Fields[identity(0x9810)] = refs(root)
	graph.Entities[returned.ID] = returned
	if _, err := CertifyPureFunction(graph); err == nil || !strings.Contains(err.Error(), "wasm.pure_expression_cycle_or_size") {
		t.Fatalf("budget error = %v", err)
	}
}

func TestPureCertificateRejectsInvalidLexicalLocals(t *testing.T) {
	tests := []struct {
		name string
		want string
		edit func(map[wire.ID]wire.Entity)
	}{
		{"initializer self read", "wasm.pure_local_read_scope", func(entities map[wire.ID]wire.Entity) {
			binding := entities[identity(0x2200)]
			binding.Fields[identity(0x9d02)] = ref(identity(0x2202))
			entities[binding.ID] = binding
		}},
		{"duplicate binding placement", "wasm.pure_local_binding_reference", func(entities map[wire.ID]wire.Entity) {
			block := entities[identity(0x2002)]
			block.Fields[identity(0x9800)] = refs(identity(0x2201), identity(0x2201), identity(0x2003))
			entities[block.ID] = block
		}},
		{"binding after terminal", "wasm.pure_local_terminal_order", func(entities map[wire.ID]wire.Entity) {
			block := entities[identity(0x2002)]
			block.Fields[identity(0x9800)] = refs(identity(0x2003), identity(0x2201))
			entities[block.ID] = block
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			graph := certifiedLocalTestGraph()
			test.edit(graph.Entities)
			if _, err := CertifyPureFunction(graph); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestPureCertificateLowersNestedIfAndUTF8Text(t *testing.T) {
	graph := certifiedTestGraph()
	function := graph.Entities[identity(0x2001)]
	function.Fields[identity(0x9112)] = ref(identity(0x2010))
	graph.Entities[function.ID] = function
	thenBlock, elseBlock := identity(0x2100), identity(0x2101)
	thenReturn, elseReturn := identity(0x2102), identity(0x2103)
	ifID, equalID, concatID := identity(0x2104), identity(0x2105), identity(0x2106)
	leftText, suffixText, rightText := identity(0x2107), identity(0x2108), identity(0x2109)
	trueID, falseID := identity(0x2110), identity(0x2111)
	graph.Entities[identity(0x2002)] = wire.Entity{ID: identity(0x2002), Schema: identity(0x9080), Fields: map[wire.ID]wire.Value{identity(0x9800): refs(ifID)}}
	graph.Entities[ifID] = wire.Entity{ID: ifID, Schema: identity(0x90c0), Fields: map[wire.ID]wire.Value{
		identity(0x9c00): ref(equalID), identity(0x9c01): ref(thenBlock), identity(0x9c02): ref(elseBlock),
	}}
	graph.Entities[thenBlock] = wire.Entity{ID: thenBlock, Schema: identity(0x9080), Fields: map[wire.ID]wire.Value{identity(0x9800): refs(thenReturn)}}
	graph.Entities[elseBlock] = wire.Entity{ID: elseBlock, Schema: identity(0x9080), Fields: map[wire.ID]wire.Value{identity(0x9800): refs(elseReturn)}}
	graph.Entities[thenReturn] = wire.Entity{ID: thenReturn, Schema: identity(0x9081), Fields: map[wire.ID]wire.Value{identity(0x9810): refs(trueID)}}
	graph.Entities[elseReturn] = wire.Entity{ID: elseReturn, Schema: identity(0x9081), Fields: map[wire.ID]wire.Value{identity(0x9810): refs(falseID)}}
	graph.Entities[equalID] = wire.Entity{ID: equalID, Schema: identity(0x90c2), Fields: map[wire.ID]wire.Value{identity(0x9c20): ref(concatID), identity(0x9c21): ref(rightText)}}
	graph.Entities[concatID] = wire.Entity{ID: concatID, Schema: identity(0x90c3), Fields: map[wire.ID]wire.Value{identity(0x9c30): ref(leftText), identity(0x9c31): ref(suffixText)}}
	graph.Entities[leftText] = wire.Entity{ID: leftText, Schema: identity(0x9050), Fields: map[wire.ID]wire.Value{identity(0x9500): byteValue("λ")}}
	graph.Entities[suffixText] = wire.Entity{ID: suffixText, Schema: identity(0x9050), Fields: map[wire.ID]wire.Value{identity(0x9500): byteValue("!")}}
	graph.Entities[rightText] = wire.Entity{ID: rightText, Schema: identity(0x9050), Fields: map[wire.ID]wire.Value{identity(0x9500): byteValue("λ!")}}
	graph.Entities[trueID] = wire.Entity{ID: trueID, Schema: identity(0x90b0), Fields: map[wire.ID]wire.Value{identity(0x9b00): {Tag: 2}}}
	graph.Entities[falseID] = wire.Entity{ID: falseID, Schema: identity(0x90b0), Fields: map[wire.ID]wire.Value{identity(0x9b00): {Tag: 1}}}
	if _, err := CertifyPureFunction(graph); err != nil {
		t.Fatal(err)
	}
	invalid := graph.Entities[leftText]
	invalid.Fields[identity(0x9500)] = wire.Value{Tag: 5, Bytes: []byte{0xff}}
	graph.Entities[leftText] = invalid
	if _, err := CertifyPureFunction(graph); err == nil || !strings.Contains(err.Error(), "wasm.pure_string_literal_type") {
		t.Fatalf("invalid UTF-8 error = %v", err)
	}
}

func certifiedTestGraph() wire.Envelope {
	programID, functionID := identity(0x2000), identity(0x2001)
	blockID, returnID := identity(0x2002), identity(0x2003)
	leftParameter, rightParameter := identity(0x2004), identity(0x2005)
	leftRead, addID, integerType := identity(0x2006), identity(0x2007), identity(0x2008)
	rightRead, booleanType := identity(0x2009), identity(0x2010)
	entities := map[wire.ID]wire.Entity{}
	entities[programID] = wire.Entity{ID: programID, Schema: identity(0x9015), Fields: map[wire.ID]wire.Value{
		identity(0x9150): refs(functionID), identity(0x9151): ref(functionID),
	}}
	entities[functionID] = wire.Entity{ID: functionID, Schema: identity(0x9011), Fields: map[wire.ID]wire.Value{
		identity(0x9110): byteValue("Combine"), identity(0x9111): refs(leftParameter, rightParameter),
		identity(0x9112): ref(integerType), identity(0x9113): ref(blockID),
	}}
	entities[blockID] = wire.Entity{ID: blockID, Schema: identity(0x9080), Fields: map[wire.ID]wire.Value{identity(0x9800): refs(returnID)}}
	entities[returnID] = wire.Entity{ID: returnID, Schema: identity(0x9081), Fields: map[wire.ID]wire.Value{identity(0x9810): refs(addID)}}
	entities[leftParameter] = testParameter(leftParameter, "left", integerType, 0)
	entities[rightParameter] = testParameter(rightParameter, "right", integerType, 1)
	entities[leftRead] = wire.Entity{ID: leftRead, Schema: identity(0x9013), Fields: map[wire.ID]wire.Value{identity(0x9130): ref(leftParameter)}}
	entities[rightRead] = wire.Entity{ID: rightRead, Schema: identity(0x9013), Fields: map[wire.ID]wire.Value{identity(0x9130): ref(rightParameter)}}
	entities[addID] = wire.Entity{ID: addID, Schema: identity(0x9014), Fields: map[wire.ID]wire.Value{
		identity(0x9140): ref(leftRead), identity(0x9141): ref(rightRead), identity(0x9142): ref(integerType),
	}}
	entities[integerType] = wire.Entity{ID: integerType, Schema: identity(0x9010), Fields: map[wire.ID]wire.Value{
		identity(0x9100): unsigned(64), identity(0x9101): {Tag: 2}, identity(0x9102): unsigned(0),
	}}
	entities[booleanType] = wire.Entity{ID: booleanType, Schema: identity(0x9020), Fields: map[wire.ID]wire.Value{}}
	return wire.Envelope{Entities: entities}
}

func certifiedLocalTestGraph() wire.Envelope {
	graph := certifiedTestGraph()
	bindingID, bindID, localReadID := identity(0x2200), identity(0x2201), identity(0x2202)
	integerType, leftRead, rightRead, addID := identity(0x2008), identity(0x2006), identity(0x2009), identity(0x2007)
	graph.Entities[bindingID] = wire.Entity{ID: bindingID, Schema: identity(0x90d0), Fields: map[wire.ID]wire.Value{
		identity(0x9d00): byteValue("total"), identity(0x9d01): ref(integerType), identity(0x9d02): ref(leftRead),
	}}
	graph.Entities[bindID] = wire.Entity{ID: bindID, Schema: identity(0x90d1), Fields: map[wire.ID]wire.Value{identity(0x9d10): ref(bindingID)}}
	graph.Entities[localReadID] = wire.Entity{ID: localReadID, Schema: identity(0x90d2), Fields: map[wire.ID]wire.Value{identity(0x9d20): ref(bindingID)}}
	add := graph.Entities[addID]
	add.Fields[identity(0x9140)] = ref(localReadID)
	add.Fields[identity(0x9141)] = ref(rightRead)
	graph.Entities[addID] = add
	block := graph.Entities[identity(0x2002)]
	block.Fields[identity(0x9800)] = refs(bindID, identity(0x2003))
	graph.Entities[block.ID] = block
	return graph
}

func testParameter(id wire.ID, name string, valueType wire.ID, index uint64) wire.Entity {
	return wire.Entity{ID: id, Schema: identity(0x9012), Fields: map[wire.ID]wire.Value{
		identity(0x9120): byteValue(name), identity(0x9121): ref(valueType), identity(0x9122): unsigned(index),
	}}
}

func ref(id wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: id} }
func refs(ids ...wire.ID) wire.Value {
	values := make([]wire.Value, len(ids))
	for index, id := range ids {
		values[index] = ref(id)
	}
	return wire.Value{Tag: 7, List: values}
}
func unsigned(value uint64) wire.Value  { return wire.Value{Tag: 3, Unsigned: value} }
func byteValue(value string) wire.Value { return wire.Value{Tag: 5, Bytes: []byte(value)} }
