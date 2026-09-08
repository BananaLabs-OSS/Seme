package wasmtarget

import (
	"bytes"
	"fmt"

	"seme.local/reference/wire"
)

type closurePlan struct {
	id          wire.ID
	typeID      wire.ID
	parameter   wire.ID
	capture     wire.ID
	environment []byte
	body        wire.ID
}

type closureValue struct {
	scalar  []byte
	closure *closurePlan
}

type closureLowering struct {
	graph     wire.Envelope
	budget    int
	stack     map[wire.ID]bool
	templates map[wire.ID]*closurePlan
}

func certifyPureClosureFunction(graph wire.Envelope, program, function wire.Entity) ([]byte, PureABI, error) {
	parameters, err := field(function, 0x9111)
	result, resultErr := field(function, 0x9112)
	body, bodyErr := field(function, 0x9113)
	if err != nil || resultErr != nil || bodyErr != nil || parameters.Tag != 7 || body.Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.closure_function_shape")
	}
	bindings := make(map[wire.ID]closureValue, len(parameters.List))
	var parameterTypes []pureValueType
	abi := PureABI{Contract: "seme.pure-abi/v1", Provider: "seme.function-v1", Target: "wasm32-pulp-reactor-v1", Fidelity: "exact", CanonicalModule: graph.Module.String(), CanonicalRevision: graph.Revision.String(), CanonicalProgram: program.ID.String(), Function: function.ID.String()}
	for index, item := range parameters.List {
		if item.Tag != 6 {
			return nil, PureABI{}, fmt.Errorf("wasm.closure_parameter")
		}
		parameter := graph.Entities[item.Reference]
		typeValue, a := field(parameter, 0x9121)
		position, b := field(parameter, 0x9122)
		if parameter.Schema != identity(0x9012) || a != nil || b != nil || position.Unsigned != uint64(index) {
			return nil, PureABI{}, fmt.Errorf("wasm.closure_parameter")
		}
		if typeEntity := graph.Entities[typeValue.Reference]; typeEntity.Schema == identity(0xa020) {
			if err := validateUnaryI64FunctionType(graph, typeEntity); err != nil {
				return nil, PureABI{}, err
			}
			// The only externally representable closure profile is a canonical
			// closure tag followed by its single immutable i64 environment.
			bindings[parameter.ID] = closureValue{closure: &closurePlan{typeID: typeEntity.ID, environment: []byte{0x20, byte(index)}}}
			abi.Parameters = append(abi.Parameters, PureABIField{Index: uint64(index), Type: "closure:i64->i64", Offset: abi.RequestSize, Size: 16, Encoding: "closure-tag-u64-then-environment-i64"})
			abi.RequestSize += 16
			continue
		}
		valueType, typeErr := pureType(graph, typeValue.Reference)
		if typeErr != nil {
			return nil, PureABI{}, typeErr
		}
		bindings[parameter.ID] = closureValue{scalar: []byte{0x20, byte(index)}}
		parameterTypes = append(parameterTypes, valueType)
		abi.Parameters = append(abi.Parameters, PureABIField{Index: uint64(index), Type: valueType.name, Offset: abi.RequestSize, Size: valueType.size, Encoding: pureEncoding(valueType.name)})
		abi.RequestSize += valueType.size
	}
	lowering := closureLowering{graph: graph, budget: 16384, stack: map[wire.ID]bool{function.ID: true}, templates: map[wire.ID]*closurePlan{}}
	for _, construct := range bySchema(graph, 0xa023) {
		typeValue, _ := field(construct, 0xa0230)
		closureParameters, _ := field(construct, 0xa0231)
		closureCaptures, _ := field(construct, 0xa0232)
		closureBody, _ := field(construct, 0xa0233)
		if closureParameters.Tag == 7 && len(closureParameters.List) == 1 && closureCaptures.Tag == 7 && len(closureCaptures.List) == 1 {
			lowering.templates[typeValue.Reference] = &closurePlan{id: construct.ID, typeID: typeValue.Reference, parameter: closureParameters.List[0].Reference, capture: closureCaptures.List[0].Reference, body: closureBody.Reference}
		}
	}
	returned, err := singleReturnExpression(graph, body.Reference)
	if err != nil {
		return nil, PureABI{}, err
	}
	value, err := lowering.value(returned, bindings, nil)
	if err != nil {
		return nil, PureABI{}, err
	}
	resultEntity := graph.Entities[result.Reference]
	if resultEntity.Schema == identity(0xa020) {
		if value.closure == nil || value.closure.typeID != resultEntity.ID {
			return nil, PureABI{}, fmt.Errorf("wasm.closure_result_value")
		}
		return lowerReturnedClosure(graph, program, function, value.closure, abi)
	}
	resultType, typeErr := pureType(graph, result.Reference)
	if typeErr != nil || resultType.name != "i64" || value.closure != nil {
		return nil, PureABI{}, fmt.Errorf("wasm.closure_result_type")
	}
	abi.ResponseSize = 8
	abi.Result = PureABIField{Index: 0, Type: "i64", Offset: 0, Size: 8, Encoding: "little-endian-i64"}
	// Run has ordinary i64 parameters after all direct calls and closure calls
	// have been physically reduced. Other external closure layouts use the
	// specialized realization below.
	if len(parameterTypes) != len(parameters.List) {
		return lowerAppliedClosure(graph, program, function, value.scalar, abi)
	}
	wasm, err := pureModule(parameterTypes, resultType, value.scalar, abi, false)
	return wasm, abi, err
}

func validateUnaryI64FunctionType(graph wire.Envelope, functionType wire.Entity) error {
	parameters, a := field(functionType, 0xa0200)
	result, b := field(functionType, 0xa0201)
	if functionType.Schema != identity(0xa020) || a != nil || b != nil || parameters.Tag != 7 || len(parameters.List) != 1 || parameters.List[0].Tag != 6 {
		return fmt.Errorf("wasm.closure_type")
	}
	for _, id := range []wire.ID{parameters.List[0].Reference, result.Reference} {
		if scalar, err := pureType(graph, id); err != nil || scalar.name != "i64" {
			return fmt.Errorf("wasm.closure_type")
		}
	}
	return nil
}

func (lowering *closureLowering) value(id wire.ID, bindings map[wire.ID]closureValue, captures map[wire.ID][]byte) (closureValue, error) {
	if lowering.budget == 0 {
		return closureValue{}, fmt.Errorf("wasm.closure_size")
	}
	lowering.budget--
	expression, ok := lowering.graph.Entities[id]
	if !ok {
		return closureValue{}, fmt.Errorf("wasm.closure_expression")
	}
	switch expression.Schema {
	case identity(0x9013):
		binding, err := field(expression, 0x9130)
		value, exists := bindings[binding.Reference]
		if err != nil || !exists {
			return closureValue{}, fmt.Errorf("wasm.closure_parameter_read")
		}
		return value, nil
	case identity(0xa022):
		binding, err := field(expression, 0xa0220)
		code, exists := captures[binding.Reference]
		if err != nil || !exists {
			return closureValue{}, fmt.Errorf("wasm.closure_capture_read")
		}
		return closureValue{scalar: append([]byte(nil), code...)}, nil
	case identity(0x9014), identity(0x9090), identity(0x90a0):
		var leftField, rightField uint64
		var opcode byte
		switch expression.Schema {
		case identity(0x9014):
			leftField, rightField, opcode = 0x9140, 0x9141, 0x7c
		case identity(0x9090):
			leftField, rightField, opcode = 0x9900, 0x9901, 0x7e
		default:
			leftField, rightField, opcode = 0x9a00, 0x9a01, 0x7d
		}
		left, _ := field(expression, leftField)
		right, _ := field(expression, rightField)
		a, err := lowering.value(left.Reference, bindings, captures)
		if err != nil {
			return closureValue{}, err
		}
		b, err := lowering.value(right.Reference, bindings, captures)
		if err != nil || a.closure != nil || b.closure != nil {
			return closureValue{}, fmt.Errorf("wasm.closure_arithmetic")
		}
		return closureValue{scalar: append(append(a.scalar, b.scalar...), opcode)}, nil
	case identity(0x9060):
		calleeValue, a := field(expression, 0x9600)
		arguments, b := field(expression, 0x9601)
		callee := lowering.graph.Entities[calleeValue.Reference]
		if a != nil || b != nil || callee.Schema != identity(0x9011) || lowering.stack[callee.ID] {
			return closureValue{}, fmt.Errorf("wasm.closure_direct_call")
		}
		calleeParameters, _ := field(callee, 0x9111)
		calleeBody, _ := field(callee, 0x9113)
		if arguments.Tag != 7 || calleeParameters.Tag != 7 || len(arguments.List) != len(calleeParameters.List) {
			return closureValue{}, fmt.Errorf("wasm.closure_call_arity")
		}
		next := map[wire.ID]closureValue{}
		for index, item := range arguments.List {
			argument, err := lowering.value(item.Reference, bindings, captures)
			if err != nil {
				return closureValue{}, err
			}
			next[calleeParameters.List[index].Reference] = argument
		}
		returned, err := singleReturnExpression(lowering.graph, calleeBody.Reference)
		if err != nil {
			return closureValue{}, err
		}
		lowering.stack[callee.ID] = true
		out, err := lowering.value(returned, next, nil)
		delete(lowering.stack, callee.ID)
		return out, err
	case identity(0xa023):
		typeValue, _ := field(expression, 0xa0230)
		parameters, _ := field(expression, 0xa0231)
		captureValues, _ := field(expression, 0xa0232)
		body, _ := field(expression, 0xa0233)
		if err := validateUnaryI64FunctionType(lowering.graph, lowering.graph.Entities[typeValue.Reference]); err != nil {
			return closureValue{}, err
		}
		if parameters.Tag != 7 || len(parameters.List) != 1 || parameters.List[0].Tag != 6 || captureValues.Tag != 7 || len(captureValues.List) != 1 || captureValues.List[0].Tag != 6 {
			return closureValue{}, fmt.Errorf("wasm.closure_shape")
		}
		parameter := lowering.graph.Entities[parameters.List[0].Reference]
		parameterType, parameterTypeErr := field(parameter, 0x9121)
		parameterPosition, parameterPositionErr := field(parameter, 0x9122)
		if parameter.Schema != identity(0x9012) || parameterTypeErr != nil || parameterPositionErr != nil || parameterPosition.Unsigned != 0 {
			return closureValue{}, fmt.Errorf("wasm.closure_parameter_shape")
		}
		if scalar, err := pureType(lowering.graph, parameterType.Reference); err != nil || scalar.name != "i64" {
			return closureValue{}, fmt.Errorf("wasm.closure_parameter_shape")
		}
		capture := lowering.graph.Entities[captureValues.List[0].Reference]
		captureType, captureTypeErr := field(capture, 0xa0211)
		captured, capturedErr := field(capture, 0xa0212)
		if capture.Schema != identity(0xa021) || captureTypeErr != nil || capturedErr != nil || captured.Tag != 6 {
			return closureValue{}, fmt.Errorf("wasm.closure_environment")
		}
		if scalar, err := pureType(lowering.graph, captureType.Reference); err != nil || scalar.name != "i64" {
			return closureValue{}, fmt.Errorf("wasm.closure_environment")
		}
		environment, err := lowering.value(captured.Reference, bindings, captures)
		if err != nil || environment.closure != nil {
			return closureValue{}, fmt.Errorf("wasm.closure_environment")
		}
		return closureValue{closure: &closurePlan{id: expression.ID, typeID: typeValue.Reference, parameter: parameters.List[0].Reference, capture: capture.ID, environment: environment.scalar, body: body.Reference}}, nil
	case identity(0xa024):
		calleeValue, _ := field(expression, 0xa0240)
		arguments, _ := field(expression, 0xa0241)
		callee, err := lowering.value(calleeValue.Reference, bindings, captures)
		if err != nil || callee.closure == nil || arguments.Tag != 7 || len(arguments.List) != 1 {
			return closureValue{}, fmt.Errorf("wasm.closure_indirect_call")
		}
		argument, err := lowering.value(arguments.List[0].Reference, bindings, captures)
		if err != nil || argument.closure != nil {
			return closureValue{}, fmt.Errorf("wasm.closure_indirect_argument")
		}
		if callee.closure.body == (wire.ID{}) {
			template, exists := lowering.templates[callee.closure.typeID]
			if !exists {
				return closureValue{}, fmt.Errorf("wasm.closure_external_call")
			}
			callee.closure = &closurePlan{id: template.id, typeID: template.typeID, parameter: template.parameter, capture: template.capture, environment: callee.closure.environment, body: template.body}
		}
		return lowering.value(callee.closure.body, map[wire.ID]closureValue{callee.closure.parameter: argument}, map[wire.ID][]byte{callee.closure.capture: callee.closure.environment})
	default:
		return closureValue{}, fmt.Errorf("wasm.closure_expression:%s", expression.Schema.String())
	}
}

func lowerReturnedClosure(graph wire.Envelope, program, function wire.Entity, closure *closurePlan, abi PureABI) ([]byte, PureABI, error) {
	abi.ResponseSize = 16
	abi.Result = PureABIField{Index: 0, Type: "closure:i64->i64", Offset: 0, Size: 16, Encoding: "closure-tag-u64-then-environment-i64"}
	return pureClosureReturnModule(closure.environment, abi)
}
func lowerAppliedClosure(graph wire.Envelope, program, function wire.Entity, code []byte, abi PureABI) ([]byte, PureABI, error) {
	abi.ResponseSize = 8
	abi.Result = PureABIField{Index: 0, Type: "i64", Offset: 0, Size: 8, Encoding: "little-endian-i64"}
	return pureInterfaceModule([]interfaceDispatchCase{{code: code}}, abi)
}

func pureClosureReturnModule(environment []byte, abi PureABI) ([]byte, PureABI, error) {
	if len(environment) == 0 {
		return nil, PureABI{}, fmt.Errorf("wasm.closure_environment")
	}
	var wasm bytes.Buffer
	wasm.Write([]byte{'\x00', 'a', 's', 'm', '\x01', 0, 0, 0})
	var types bytes.Buffer
	uleb(&types, 6)
	functionType(&types, []byte{0x7f}, []byte{0x7f})
	functionType(&types, []byte{0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f}, []byte{0x7f})
	functionType(&types, nil, []byte{0x7f})
	functionType(&types, []byte{0x7f, 0x7f}, []byte{0x7f})
	functionType(&types, []byte{0x7e}, []byte{0x7e})
	functionType(&types, []byte{0x7f, 0x7f}, nil)
	section(&wasm, 1, types.Bytes())
	section(&wasm, 3, []byte{6, 0, 5, 3, 3, 2, 1})
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
	export(&exports, "pulp_on_call", 0, 5)
	section(&wasm, 7, exports.Bytes())
	var provider bytes.Buffer
	provider.Write([]byte{1, 1, 0x7e, 0x20, 3, 0x41, 8, 0x47, 0x04, 0x40, 0x41, 2, 0x0f, 0x0b, 0x20, 2, 0x29, 3, 0, 0x21, 6})
	constI32(&provider, 8192)
	provider.WriteByte(0x42)
	sleb(&provider, 1)
	provider.Write([]byte{0x37, 3, 0})
	constI32(&provider, 8192)
	provider.Write([]byte{0x20, 6, 0x37, 3, 8, 0x20, 4})
	constI32(&provider, 8192)
	provider.Write([]byte{0x36, 2, 0, 0x20, 5, 0x41, 16, 0x36, 2, 0, 0x41, 0, 0x0b})
	bodies := [][]byte{pureAllocatorBody(), pureFreeBody(), {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}, {0, 0x41, 0, 0x0b}, provider.Bytes()}
	var code bytes.Buffer
	uleb(&code, uint64(len(bodies)))
	for _, body := range bodies {
		uleb(&code, uint64(len(body)))
		code.Write(body)
	}
	section(&wasm, 10, code.Bytes())
	return wasm.Bytes(), abi, nil
}
