package wasmtarget

import (
	"bytes"
	"fmt"

	"seme.local/reference/wire"
)

// certifyPureTransitionFunction realizes the first bounded, value-semantic
// state-transition ABI. Records are passed and returned by value; the source
// receiver is never exposed as writable storage.
func certifyPureTransitionFunction(graph wire.Envelope, program, function wire.Entity) ([]byte, PureABI, error) {
	parameters, err := field(function, 0x9111)
	if err != nil || parameters.Tag != 7 || len(parameters.List) != 2 {
		return nil, PureABI{}, fmt.Errorf("wasm.transition_parameters")
	}
	parameterIDs := make([]wire.ID, 2)
	parameterTypes := map[wire.ID]string{}
	for index, value := range parameters.List {
		if value.Tag != 6 {
			return nil, PureABI{}, fmt.Errorf("wasm.transition_parameter")
		}
		parameter := graph.Entities[value.Reference]
		position, pErr := field(parameter, 0x9122)
		typeValue, tErr := field(parameter, 0x9121)
		if parameter.Schema != identity(0x9012) || pErr != nil || tErr != nil || position.Tag != 3 || position.Unsigned != uint64(index) || typeValue.Tag != 6 {
			return nil, PureABI{}, fmt.Errorf("wasm.transition_parameter")
		}
		parameterIDs[index] = parameter.ID
		parameterTypes[parameter.ID] = "i64"
		if index == 0 {
			if err := validateSingleI64Record(graph, typeValue.Reference); err != nil {
				return nil, PureABI{}, err
			}
		} else if valueType, typeErr := pureType(graph, typeValue.Reference); typeErr != nil || valueType.name != "i64" {
			return nil, PureABI{}, fmt.Errorf("wasm.transition_delta_type")
		}
	}
	resultValue, _ := field(function, 0x9112)
	transitionType := graph.Entities[resultValue.Reference]
	stateType, a := field(transitionType, 0xa0040)
	valueType, b := field(transitionType, 0xa0041)
	if a != nil || b != nil || stateType.Tag != 6 || valueType.Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.transition_type")
	}
	if err := validateSingleI64Record(graph, stateType.Reference); err != nil {
		return nil, PureABI{}, err
	}
	if scalar, scalarErr := pureType(graph, valueType.Reference); scalarErr != nil || scalar.name != "i64" {
		return nil, PureABI{}, fmt.Errorf("wasm.transition_result_type")
	}
	body, _ := field(function, 0x9113)
	block := graph.Entities[body.Reference]
	statements, blockErr := field(block, 0x9800)
	if body.Tag != 6 || block.Schema != identity(0x9080) || blockErr != nil || statements.Tag != 7 || len(statements.List) != 1 {
		return nil, PureABI{}, fmt.Errorf("wasm.transition_entry_body")
	}
	ret := graph.Entities[statements.List[0].Reference]
	values, returnErr := field(ret, 0x9810)
	if ret.Schema != identity(0x9081) || returnErr != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.transition_entry_return")
	}
	call := graph.Entities[values.List[0].Reference]
	receiver, rErr := field(call, 0xa0030)
	methodValue, mErr := field(call, 0xa0031)
	arguments, argsErr := field(call, 0xa0032)
	if call.Schema != identity(0xa003) || rErr != nil || mErr != nil || argsErr != nil || receiver.Tag != 6 || methodValue.Tag != 6 || arguments.Tag != 7 || len(arguments.List) != 1 || arguments.List[0].Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.transition_method_call")
	}
	method := graph.Entities[methodValue.Reference]
	receiverBinding, rbErr := field(method, 0xa0021)
	methodParameters, mpErr := field(method, 0xa0022)
	methodResult, mrErr := field(method, 0xa0023)
	methodBody, mbErr := field(method, 0xa0024)
	if method.Schema != identity(0xa002) || rbErr != nil || mpErr != nil || mrErr != nil || mbErr != nil || receiverBinding.Tag != 6 || methodParameters.Tag != 7 || len(methodParameters.List) != 1 || methodParameters.List[0].Tag != 6 || methodResult.Reference != transitionType.ID || methodBody.Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.transition_method")
	}
	if graph.Entities[receiverBinding.Reference].Schema != identity(0xa000) {
		return nil, PureABI{}, fmt.Errorf("wasm.transition_receiver")
	}
	receiverType, receiverTypeErr := field(graph.Entities[receiverBinding.Reference], 0xa0001)
	methodParameterEntity := graph.Entities[methodParameters.List[0].Reference]
	methodParameterType, methodParameterTypeErr := field(methodParameterEntity, 0x9121)
	methodParameterPosition, methodParameterPositionErr := field(methodParameterEntity, 0x9122)
	if receiverTypeErr != nil || receiverType.Reference != stateType.Reference || methodParameterEntity.Schema != identity(0x9012) || methodParameterTypeErr != nil || methodParameterPositionErr != nil || methodParameterPosition.Unsigned != 0 {
		return nil, PureABI{}, fmt.Errorf("wasm.transition_method_binding")
	}
	if scalar, scalarErr := pureType(graph, methodParameterType.Reference); scalarErr != nil || scalar.name != "i64" {
		return nil, PureABI{}, fmt.Errorf("wasm.transition_method_binding")
	}
	// The bounded profile requires the receiver and argument to originate from
	// the entry parameters. This also makes receiver aliasing impossible.
	entryLocals := map[wire.ID]byte{parameterIDs[0]: 0, parameterIDs[1]: 1}
	receiverCode, err := lowerTransitionI64(graph, receiver.Reference, entryLocals, nil, map[wire.ID]bool{})
	if err != nil {
		return nil, PureABI{}, err
	}
	argumentCode, err := lowerTransitionI64(graph, arguments.List[0].Reference, entryLocals, nil, map[wire.ID]bool{})
	if err != nil {
		return nil, PureABI{}, err
	}
	if !bytes.Equal(receiverCode, []byte{0x20, 0}) || !bytes.Equal(argumentCode, []byte{0x20, 1}) {
		return nil, PureABI{}, fmt.Errorf("wasm.transition_entry_binding")
	}
	methodParameter := methodParameters.List[0].Reference
	methodGraph, err := normalizePureLocals(graph, methodBody.Reference, map[wire.ID]string{methodParameter: "i64"})
	if err != nil {
		return nil, PureABI{}, err
	}
	methodBlock := methodGraph.Entities[methodBody.Reference]
	methodStatements, _ := field(methodBlock, 0x9800)
	methodReturn := methodGraph.Entities[methodStatements.List[0].Reference]
	methodValues, _ := field(methodReturn, 0x9810)
	if methodReturn.Schema != identity(0x9081) || methodValues.Tag != 7 || len(methodValues.List) != 1 || methodValues.List[0].Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.transition_method_return")
	}
	transition := methodGraph.Entities[methodValues.List[0].Reference]
	state, sErr := field(transition, 0xa0051)
	result, outErr := field(transition, 0xa0052)
	typeRef, typeErr := field(transition, 0xa0050)
	if transition.Schema != identity(0xa005) || sErr != nil || outErr != nil || typeErr != nil || typeRef.Reference != transitionType.ID {
		return nil, PureABI{}, fmt.Errorf("wasm.transition_value")
	}
	locals := map[wire.ID]byte{methodParameter: 1}
	stateCode, err := lowerTransitionI64(methodGraph, state.Reference, locals, &receiverBinding.Reference, map[wire.ID]bool{})
	if err != nil {
		return nil, PureABI{}, err
	}
	resultCode, err := lowerTransitionI64(methodGraph, result.Reference, locals, &receiverBinding.Reference, map[wire.ID]bool{})
	if err != nil {
		return nil, PureABI{}, err
	}
	abi := PureABI{Contract: "seme.pure-abi/v1", Provider: "seme.function-v1", Target: "wasm32-pulp-reactor-v1", Fidelity: "exact", CanonicalModule: graph.Module.String(), CanonicalRevision: graph.Revision.String(), CanonicalProgram: program.ID.String(), Function: function.ID.String(), RequestSize: 16, ResponseSize: 16,
		Parameters: []PureABIField{{Index: 0, Type: "record:i64", Offset: 0, Size: 8, Encoding: "little-endian-i64"}, {Index: 1, Type: "i64", Offset: 8, Size: 8, Encoding: "little-endian-i64"}},
		Result:     PureABIField{Index: 0, Type: "state-transition:record:i64,i64", Offset: 0, Size: 16, Encoding: "state-then-result-little-endian-i64"}}
	wasm, err := pureTransitionModule(stateCode, resultCode, abi)
	return wasm, abi, err
}

func validateSingleI64Record(graph wire.Envelope, id wire.ID) error {
	record := graph.Entities[id]
	fields, err := field(record, 0x9301)
	if record.Schema != identity(0x9030) || err != nil || fields.Tag != 7 || len(fields.List) != 1 || fields.List[0].Tag != 6 {
		return fmt.Errorf("wasm.transition_state_type")
	}
	member := graph.Entities[fields.List[0].Reference]
	typeValue, a := field(member, 0x9311)
	position, b := field(member, 0x9312)
	if member.Schema != identity(0x9031) || a != nil || b != nil || position.Tag != 3 || position.Unsigned != 0 {
		return fmt.Errorf("wasm.transition_state_field")
	}
	if scalar, scalarErr := pureType(graph, typeValue.Reference); scalarErr != nil || scalar.name != "i64" {
		return fmt.Errorf("wasm.transition_state_field")
	}
	return nil
}

func lowerTransitionI64(graph wire.Envelope, id wire.ID, parameters map[wire.ID]byte, receiver *wire.ID, visiting map[wire.ID]bool) ([]byte, error) {
	if visiting[id] {
		return nil, fmt.Errorf("wasm.transition_expression_cycle")
	}
	visiting[id] = true
	defer delete(visiting, id)
	expression, ok := graph.Entities[id]
	if !ok {
		return nil, fmt.Errorf("wasm.transition_expression")
	}
	switch expression.Schema {
	case identity(0x9013):
		parameter, err := field(expression, 0x9130)
		local, exists := parameters[parameter.Reference]
		if err != nil || !exists {
			return nil, fmt.Errorf("wasm.transition_parameter_read")
		}
		return []byte{0x20, local}, nil
	case identity(0xa001):
		binding, err := field(expression, 0xa0010)
		if err != nil || receiver == nil || binding.Reference != *receiver {
			return nil, fmt.Errorf("wasm.transition_receiver_read")
		}
		return []byte{0x20, 0}, nil
	case identity(0x9032):
		record, a := field(expression, 0x9320)
		member, b := field(expression, 0x9321)
		if a != nil || b != nil || member.Tag != 6 {
			return nil, fmt.Errorf("wasm.transition_field_read")
		}
		fieldEntity := graph.Entities[member.Reference]
		position, pErr := field(fieldEntity, 0x9312)
		if fieldEntity.Schema != identity(0x9031) || pErr != nil || position.Unsigned != 0 {
			return nil, fmt.Errorf("wasm.transition_field")
		}
		if construct, exists := graph.Entities[record.Reference]; exists && construct.Schema == identity(0x9033) {
			typeValue, typeErr := field(construct, 0x9330)
			typeEntity := graph.Entities[typeValue.Reference]
			members, membersErr := field(typeEntity, 0x9301)
			if typeErr != nil || membersErr != nil || members.Tag != 7 || len(members.List) != 1 || members.List[0].Reference != member.Reference {
				return nil, fmt.Errorf("wasm.transition_field_membership")
			}
		}
		return lowerTransitionI64(graph, record.Reference, parameters, receiver, visiting)
	case identity(0x9033):
		typeValue, typeErr := field(expression, 0x9330)
		values, err := field(expression, 0x9331)
		if typeErr != nil || validateSingleI64Record(graph, typeValue.Reference) != nil || err != nil || values.Tag != 7 || len(values.List) != 1 || values.List[0].Tag != 6 {
			return nil, fmt.Errorf("wasm.transition_record")
		}
		return lowerTransitionI64(graph, values.List[0].Reference, parameters, receiver, visiting)
	case identity(0x9014):
		left, a := field(expression, 0x9140)
		right, b := field(expression, 0x9141)
		if a != nil || b != nil {
			return nil, fmt.Errorf("wasm.transition_add")
		}
		leftCode, err := lowerTransitionI64(graph, left.Reference, parameters, receiver, visiting)
		if err != nil {
			return nil, err
		}
		rightCode, err := lowerTransitionI64(graph, right.Reference, parameters, receiver, visiting)
		if err != nil {
			return nil, err
		}
		return append(append(leftCode, rightCode...), 0x7c), nil
	default:
		return nil, fmt.Errorf("wasm.transition_expression:%s", expression.Schema.String())
	}
}

func pureTransitionModule(stateCode, resultCode []byte, abi PureABI) ([]byte, error) {
	if len(stateCode) == 0 || len(resultCode) == 0 {
		return nil, fmt.Errorf("wasm.transition_layout")
	}
	var wasm bytes.Buffer
	wasm.Write([]byte{'\x00', 'a', 's', 'm', '\x01', 0, 0, 0})
	var types bytes.Buffer
	uleb(&types, 6)
	functionType(&types, []byte{0x7f}, []byte{0x7f})
	functionType(&types, []byte{0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f}, []byte{0x7f})
	functionType(&types, nil, []byte{0x7f})
	functionType(&types, []byte{0x7f, 0x7f}, []byte{0x7f})
	functionType(&types, []byte{0x7e, 0x7e}, []byte{0x7e})
	functionType(&types, []byte{0x7f, 0x7f}, nil)
	section(&wasm, 1, types.Bytes())
	section(&wasm, 3, []byte{8, 0, 5, 3, 3, 2, 4, 4, 1})
	section(&wasm, 5, []byte{1, 0, 1})
	var globals bytes.Buffer
	globals.Write([]byte{1, 0x7f, 1, 0x41})
	sleb(&globals, 1024)
	globals.WriteByte(0x0b)
	section(&wasm, 6, globals.Bytes())
	var exports bytes.Buffer
	uleb(&exports, 7)
	export(&exports, "memory", 2, 0)
	export(&exports, "pulp_alloc", 0, 0)
	export(&exports, "pulp_free", 0, 1)
	export(&exports, "pulp_init", 0, 2)
	export(&exports, "pulp_step", 0, 3)
	export(&exports, "pulp_shutdown", 0, 4)
	export(&exports, "pulp_on_call", 0, 7)
	section(&wasm, 7, exports.Bytes())
	helper := func(code []byte) []byte { return append(append([]byte{0}, code...), 0x0b) }
	var provider bytes.Buffer
	provider.Write([]byte{1, 2, 0x7e})
	provider.Write([]byte{0x20, 3, 0x41, 0x10, 0x47, 0x04, 0x40, 0x41, 2, 0x0f, 0x0b})
	provider.Write([]byte{0x20, 2, 0x29, 3, 0, 0x20, 2, 0x29, 3, 8, 0x10, 5, 0x21, 6, 0x20, 2, 0x29, 3, 0, 0x20, 2, 0x29, 3, 8, 0x10, 6, 0x21, 7})
	constI32(&provider, 8192)
	provider.Write([]byte{0x20, 6, 0x37, 3, 0})
	constI32(&provider, 8192)
	provider.Write([]byte{0x20, 7, 0x37, 3, 8, 0x20, 4})
	constI32(&provider, 8192)
	provider.Write([]byte{0x36, 2, 0, 0x20, 5, 0x41, 0x10, 0x36, 2, 0, 0x41, 0, 0x0b})
	bodies := [][]byte{pureAllocatorBody(), pureFreeBody(), {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}, helper(stateCode), helper(resultCode), provider.Bytes()}
	var code bytes.Buffer
	uleb(&code, uint64(len(bodies)))
	for _, body := range bodies {
		uleb(&code, uint64(len(body)))
		code.Write(body)
	}
	section(&wasm, 10, code.Bytes())
	return wasm.Bytes(), nil
}
