package wasmtarget

import (
	"fmt"

	"seme.local/reference/wire"
)

func certifyPureMutableClosureFunction(graph wire.Envelope, program, function wire.Entity) ([]byte, PureABI, error) {
	parameters, _ := field(function, 0x9111)
	result, _ := field(function, 0x9112)
	body, _ := field(function, 0x9113)
	if parameters.Tag != 7 || len(parameters.List) != 3 || result.Tag != 6 || body.Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.mutable_closure_entry")
	}
	parameterIDs := make([]wire.ID, 3)
	for index, item := range parameters.List {
		if item.Tag != 6 {
			return nil, PureABI{}, fmt.Errorf("wasm.mutable_closure_parameter")
		}
		parameter := graph.Entities[item.Reference]
		typeValue, a := field(parameter, 0x9121)
		position, b := field(parameter, 0x9122)
		scalar, e := pureType(graph, typeValue.Reference)
		if parameter.Schema != identity(0x9012) || a != nil || b != nil || e != nil || scalar.name != "i64" || position.Unsigned != uint64(index) {
			return nil, PureABI{}, fmt.Errorf("wasm.mutable_closure_parameter")
		}
		parameterIDs[index] = parameter.ID
	}
	if scalar, e := pureType(graph, result.Reference); e != nil || scalar.name != "i64" {
		return nil, PureABI{}, fmt.Errorf("wasm.mutable_closure_result")
	}
	block := graph.Entities[body.Reference]
	statements, _ := field(block, 0x9800)
	if block.Schema != identity(0x9080) || statements.Tag != 7 || len(statements.List) != 4 {
		return nil, PureABI{}, fmt.Errorf("wasm.mutable_closure_entry_body")
	}
	declarePlace := graph.Entities[statements.List[0].Reference]
	placeRef, _ := field(declarePlace, 0x9e10)
	place := graph.Entities[placeRef.Reference]
	placeType, _ := field(place, 0x9e01)
	initializer, _ := field(place, 0x9e02)
	if declarePlace.Schema != identity(0x90e1) || place.Schema != identity(0x90e0) || validateUnaryI64FunctionType(graph, graph.Entities[placeType.Reference]) != nil {
		return nil, PureABI{}, fmt.Errorf("wasm.mutable_closure_place")
	}
	makeCall := graph.Entities[initializer.Reference]
	if makeCall.Schema == identity(0xa034) {
		if err := validateMutableClosure(graph, makeCall, placeType.Reference, parameterIDs[0]); err != nil {
			return nil, PureABI{}, err
		}
	} else {
		calleeValue, calleeErr := field(makeCall, 0x9600)
		makeArgs, argsErr := field(makeCall, 0x9601)
		makeFunction := graph.Entities[calleeValue.Reference]
		if makeCall.Schema != identity(0x9060) || calleeErr != nil || argsErr != nil || makeArgs.Tag != 7 || len(makeArgs.List) != 1 || !isParameterRead(graph, makeArgs.List[0].Reference, parameterIDs[0]) {
			return nil, PureABI{}, fmt.Errorf("wasm.mutable_closure_constructor_call")
		}
		if err := validateMutableConstructor(graph, makeFunction, placeType.Reference); err != nil {
			return nil, PureABI{}, err
		}
	}
	declareFirst := graph.Entities[statements.List[1].Reference]
	firstLocalRef, _ := field(declareFirst, 0x9d10)
	firstLocal := graph.Entities[firstLocalRef.Reference]
	firstType, _ := field(firstLocal, 0x9d01)
	firstInitializer, _ := field(firstLocal, 0x9d02)
	transitionType := graph.Entities[firstType.Reference]
	transitionStateType, _ := field(transitionType, 0xa0040)
	transitionResultType, _ := field(transitionType, 0xa0041)
	if declareFirst.Schema != identity(0x90d1) || firstLocal.Schema != identity(0x90d0) || transitionType.Schema != identity(0xa004) || transitionStateType.Reference != placeType.Reference {
		return nil, PureABI{}, fmt.Errorf("wasm.mutable_closure_transition_type")
	}
	if scalar, e := pureType(graph, transitionResultType.Reference); e != nil || scalar.name != "i64" {
		return nil, PureABI{}, fmt.Errorf("wasm.mutable_closure_transition_type")
	}
	if err := validateStatefulCall(graph, firstInitializer.Reference, place.ID, parameterIDs[1]); err != nil {
		return nil, PureABI{}, err
	}
	assignment := graph.Entities[statements.List[2].Reference]
	assignedPlace, _ := field(assignment, 0x9e30)
	assignedValue, _ := field(assignment, 0x9e31)
	stateProjection := graph.Entities[assignedValue.Reference]
	stateCall, _ := field(stateProjection, 0xa0060)
	firstRead := graph.Entities[stateCall.Reference]
	firstBinding, firstBindingErr := field(firstRead, 0x9d20)
	if assignment.Schema != identity(0x90e3) || assignedPlace.Reference != place.ID || stateProjection.Schema != identity(0xa006) || firstRead.Schema != identity(0x90d2) || firstBindingErr != nil || firstBinding.Reference != firstLocal.ID {
		return nil, PureABI{}, fmt.Errorf("wasm.mutable_closure_commit")
	}
	ret := graph.Entities[statements.List[3].Reference]
	returnValues, _ := field(ret, 0x9810)
	if ret.Schema != identity(0x9081) || returnValues.Tag != 7 || len(returnValues.List) != 1 {
		return nil, PureABI{}, fmt.Errorf("wasm.mutable_closure_return")
	}
	resultProjection := graph.Entities[returnValues.List[0].Reference]
	secondCall, _ := field(resultProjection, 0xa0070)
	if resultProjection.Schema != identity(0xa007) {
		return nil, PureABI{}, fmt.Errorf("wasm.mutable_closure_return")
	}
	if err := validateStatefulCall(graph, secondCall.Reference, place.ID, parameterIDs[2]); err != nil {
		return nil, PureABI{}, err
	}
	// The committed first transition is deliberately represented in the code:
	// ((start + first) + second). No host state or aliased source storage exists.
	instructions := []byte{0x20, 0, 0x20, 1, 0x7c, 0x20, 2, 0x7c}
	i64 := pureValueType{name: "i64", wasm: 0x7e, size: 8}
	abi := PureABI{Contract: "seme.pure-abi/v1", Provider: "seme.function-v1", Target: "wasm32-pulp-reactor-v1", Fidelity: "exact", CanonicalModule: graph.Module.String(), CanonicalRevision: graph.Revision.String(), CanonicalProgram: program.ID.String(), Function: function.ID.String(), RequestSize: 24, ResponseSize: 8, Parameters: []PureABIField{{Index: 0, Type: "i64", Offset: 0, Size: 8, Encoding: pureEncoding("i64")}, {Index: 1, Type: "i64", Offset: 8, Size: 8, Encoding: pureEncoding("i64")}, {Index: 2, Type: "i64", Offset: 16, Size: 8, Encoding: pureEncoding("i64")}}, Result: PureABIField{Index: 0, Type: "i64", Offset: 0, Size: 8, Encoding: "little-endian-i64"}}
	wasm, err := pureModule([]pureValueType{i64, i64, i64}, i64, instructions, abi, false)
	return wasm, abi, err
}

func isParameterRead(graph wire.Envelope, id, parameter wire.ID) bool {
	entity := graph.Entities[id]
	binding, err := field(entity, 0x9130)
	return entity.Schema == identity(0x9013) && err == nil && binding.Reference == parameter
}
func isPlaceRead(graph wire.Envelope, id, place wire.ID) bool {
	entity := graph.Entities[id]
	binding, err := field(entity, 0x9e20)
	return entity.Schema == identity(0x90e2) && err == nil && binding.Reference == place
}

func validateStatefulCall(graph wire.Envelope, id, place, argument wire.ID) error {
	call := graph.Entities[id]
	callee, _ := field(call, 0xa0350)
	arguments, _ := field(call, 0xa0351)
	if call.Schema != identity(0xa035) || arguments.Tag != 7 || len(arguments.List) != 1 || !isPlaceRead(graph, callee.Reference, place) || !isParameterRead(graph, arguments.List[0].Reference, argument) {
		return fmt.Errorf("wasm.mutable_closure_stateful_call")
	}
	return nil
}

func validateMutableConstructor(graph wire.Envelope, function wire.Entity, functionType wire.ID) error {
	parameters, _ := field(function, 0x9111)
	result, _ := field(function, 0x9112)
	body, _ := field(function, 0x9113)
	if function.Schema != identity(0x9011) || parameters.Tag != 7 || len(parameters.List) != 1 || result.Reference != functionType {
		return fmt.Errorf("wasm.mutable_closure_constructor")
	}
	parameter := graph.Entities[parameters.List[0].Reference]
	parameterType, _ := field(parameter, 0x9121)
	if scalar, e := pureType(graph, parameterType.Reference); parameter.Schema != identity(0x9012) || e != nil || scalar.name != "i64" {
		return fmt.Errorf("wasm.mutable_closure_constructor")
	}
	returned, e := singleReturnExpression(graph, body.Reference)
	if e != nil {
		return e
	}
	closure := graph.Entities[returned]
	return validateMutableClosure(graph, closure, functionType, parameter.ID)
}

func validateMutableClosure(graph wire.Envelope, closure wire.Entity, functionType, initialParameter wire.ID) error {
	closureType, _ := field(closure, 0xa0340)
	closureParameters, _ := field(closure, 0xa0341)
	captures, _ := field(closure, 0xa0342)
	closureBody, _ := field(closure, 0xa0343)
	if closure.Schema != identity(0xa034) || closureType.Reference != functionType || closureParameters.Tag != 7 || len(closureParameters.List) != 1 || captures.Tag != 7 || len(captures.List) != 1 {
		return fmt.Errorf("wasm.mutable_closure_shape")
	}
	capture := graph.Entities[captures.List[0].Reference]
	captureType, _ := field(capture, 0xa0301)
	initial, _ := field(capture, 0xa0302)
	if capture.Schema != identity(0xa030) || !isParameterRead(graph, initial.Reference, initialParameter) {
		return fmt.Errorf("wasm.mutable_closure_environment")
	}
	if scalar, e := pureType(graph, captureType.Reference); e != nil || scalar.name != "i64" {
		return fmt.Errorf("wasm.mutable_closure_environment")
	}
	closureParameter := graph.Entities[closureParameters.List[0].Reference]
	closureParameterType, _ := field(closureParameter, 0x9121)
	if scalar, e := pureType(graph, closureParameterType.Reference); closureParameter.Schema != identity(0x9012) || e != nil || scalar.name != "i64" {
		return fmt.Errorf("wasm.mutable_closure_parameter")
	}
	sequence := graph.Entities[closureBody.Reference]
	steps, _ := field(sequence, 0xa0330)
	sequenceResult, _ := field(sequence, 0xa0331)
	if sequence.Schema != identity(0xa033) || steps.Tag != 7 || len(steps.List) != 1 {
		return fmt.Errorf("wasm.mutable_closure_sequence")
	}
	update := graph.Entities[steps.List[0].Reference]
	updatedCapture, _ := field(update, 0xa0320)
	updatedValue, _ := field(update, 0xa0321)
	readResult := graph.Entities[sequenceResult.Reference]
	readCapture, _ := field(readResult, 0xa0310)
	if update.Schema != identity(0xa032) || updatedCapture.Reference != capture.ID || readResult.Schema != identity(0xa031) || readCapture.Reference != capture.ID {
		return fmt.Errorf("wasm.mutable_closure_update")
	}
	addition := graph.Entities[updatedValue.Reference]
	left, _ := field(addition, 0x9140)
	right, _ := field(addition, 0x9141)
	leftRead := graph.Entities[left.Reference]
	leftCapture, _ := field(leftRead, 0xa0310)
	if addition.Schema != identity(0x9014) || leftRead.Schema != identity(0xa031) || leftCapture.Reference != capture.ID || !isParameterRead(graph, right.Reference, closureParameter.ID) {
		return fmt.Errorf("wasm.mutable_closure_update_value")
	}
	return nil
}
