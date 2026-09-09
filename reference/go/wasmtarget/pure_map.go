package wasmtarget

import (
	"fmt"

	"seme.local/reference/wire"
)

func certifyPureMapFunction(graph wire.Envelope, program, function wire.Entity) ([]byte, PureABI, error) {
	parameters, _ := field(function, 0x9111)
	result, _ := field(function, 0x9112)
	body, _ := field(function, 0x9113)
	if parameters.Tag != 7 || len(parameters.List) != 2 || result.Tag != 6 || body.Tag != 6 {
		return nil, PureABI{}, fmt.Errorf("wasm.map_function")
	}
	parameterIDs := make([]wire.ID, 2)
	for index, item := range parameters.List {
		if item.Tag != 6 {
			return nil, PureABI{}, fmt.Errorf("wasm.map_parameter")
		}
		parameter := graph.Entities[item.Reference]
		typeValue, a := field(parameter, 0x9121)
		position, b := field(parameter, 0x9122)
		if parameter.Schema != identity(0x9012) || a != nil || b != nil || position.Unsigned != uint64(index) {
			return nil, PureABI{}, fmt.Errorf("wasm.map_parameter")
		}
		parameterIDs[index] = parameter.ID
		if index == 0 {
			slice := graph.Entities[typeValue.Reference]
			element, e := field(slice, 0x9f80)
			if slice.Schema != identity(0x90f8) || e != nil || !isI64Type(graph, element.Reference) {
				return nil, PureABI{}, fmt.Errorf("wasm.map_collection_type")
			}
		} else if !isI64Type(graph, typeValue.Reference) {
			return nil, PureABI{}, fmt.Errorf("wasm.map_key_type")
		}
	}
	if !isI64Type(graph, result.Reference) {
		return nil, PureABI{}, fmt.Errorf("wasm.map_result_type")
	}
	returned, err := singleReturnExpression(graph, body.Reference)
	if err != nil {
		return nil, PureABI{}, err
	}
	lookup := graph.Entities[returned]
	mapping, _ := field(lookup, 0xa0420)
	query, _ := field(lookup, 0xa0421)
	if lookup.Schema != identity(0xa042) || !isParameterRead(graph, query.Reference, parameterIDs[1]) {
		return nil, PureABI{}, fmt.Errorf("wasm.map_final_lookup")
	}
	fold := graph.Entities[mapping.Reference]
	collection, _ := field(fold, 0x9f70)
	initial, _ := field(fold, 0x9f71)
	accumulator, _ := field(fold, 0x9f72)
	element, _ := field(fold, 0x9f73)
	foldBody, _ := field(fold, 0x9f74)
	if fold.Schema != identity(0x90f7) || !isParameterRead(graph, collection.Reference, parameterIDs[0]) {
		return nil, PureABI{}, fmt.Errorf("wasm.map_fold")
	}
	empty := graph.Entities[initial.Reference]
	mapTypeValue, _ := field(empty, 0xa0410)
	mapType := graph.Entities[mapTypeValue.Reference]
	keyType, _ := field(mapType, 0xa0400)
	valueType, _ := field(mapType, 0xa0401)
	if empty.Schema != identity(0xa041) || mapType.Schema != identity(0xa040) || !isI64Type(graph, keyType.Reference) || !isI64Type(graph, valueType.Reference) {
		return nil, PureABI{}, fmt.Errorf("wasm.map_type")
	}
	accumulatorBinding := graph.Entities[accumulator.Reference]
	accumulatorType, _ := field(accumulatorBinding, 0x9f51)
	elementBinding := graph.Entities[element.Reference]
	elementType, _ := field(elementBinding, 0x9f51)
	if accumulatorBinding.Schema != identity(0x90f5) || accumulatorType.Reference != mapType.ID || elementBinding.Schema != identity(0x90f5) || !isI64Type(graph, elementType.Reference) {
		return nil, PureABI{}, fmt.Errorf("wasm.map_bindings")
	}
	update := graph.Entities[foldBody.Reference]
	updateMap, _ := field(update, 0xa0430)
	updateKey, _ := field(update, 0xa0431)
	updateValue, _ := field(update, 0xa0432)
	if update.Schema != identity(0xa043) || !isIterationRead(graph, updateMap.Reference, accumulatorBinding.ID) || !isIterationRead(graph, updateKey.Reference, elementBinding.ID) {
		return nil, PureABI{}, fmt.Errorf("wasm.map_update")
	}
	addition := graph.Entities[updateValue.Reference]
	left, _ := field(addition, 0x9140)
	right, _ := field(addition, 0x9141)
	innerLookup := graph.Entities[left.Reference]
	innerMap, _ := field(innerLookup, 0xa0420)
	innerKey, _ := field(innerLookup, 0xa0421)
	literal := graph.Entities[right.Reference]
	literalValue, _ := field(literal, 0x9700)
	if addition.Schema != identity(0x9014) || innerLookup.Schema != identity(0xa042) || !isIterationRead(graph, innerMap.Reference, accumulatorBinding.ID) || !isIterationRead(graph, innerKey.Reference, elementBinding.ID) || literal.Schema != identity(0x9070) || literalValue.Unsigned != 1 {
		return nil, PureABI{}, fmt.Errorf("wasm.map_tally_body")
	}
	// Physical realization counts matching keys directly. This is equivalent to
	// constructing the immutable map with zero-on-missing lookup, while avoiding
	// observable insertion order or aliased map storage.
	helper := []byte{0x20, 0, 0xa7, 0x21, 4, 0x20, 0, 0x42, 32, 0x88, 0xa7, 0x21, 5, 0x41, 0, 0x21, 2, 0x42, 0, 0x21, 3, 0x02, 0x40, 0x03, 0x40, 0x20, 2, 0x20, 5, 0x4f, 0x0d, 1, 0x20, 4, 0x20, 2, 0x41, 3, 0x74, 0x6a, 0x29, 3, 0, 0x20, 1, 0x51, 0x04, 0x40, 0x20, 3, 0x42, 1, 0x7c, 0x21, 3, 0x0b, 0x20, 2, 0x41, 1, 0x6a, 0x21, 2, 0x0c, 0, 0x0b, 0x0b, 0x20, 3}
	slice := pureValueType{name: "slice:i64", wasm: 0x7e, size: 8}
	i64 := pureValueType{name: "i64", wasm: 0x7e, size: 8}
	abi := PureABI{Contract: "seme.pure-abi/v2", Provider: "seme.function-v2", Target: "wasm32-pulp-reactor-v1", Fidelity: "exact", CanonicalModule: graph.Module.String(), CanonicalRevision: graph.Revision.String(), CanonicalProgram: program.ID.String(), Function: function.ID.String(), RequestSize: 16, ResponseSize: 8, FixedHeaderSize: 16, MaximumRequestSize: 7160, VariablePayload: true, Parameters: []PureABIField{{Index: 0, Type: "slice:i64", Offset: 0, Size: 8, Encoding: pureEncoding("slice:i64")}, {Index: 1, Type: "i64", Offset: 8, Size: 8, Encoding: pureEncoding("i64")}}, Result: PureABIField{Index: 0, Type: "i64", Offset: 0, Size: 8, Encoding: "little-endian-i64"}}
	wasm, err := pureStringModule([]pureValueType{slice, i64}, i64, helper, abi, nil, []byte{0x7f, 0x7e, 0x7f, 0x7f})
	return wasm, abi, err
}

func isI64Type(graph wire.Envelope, id wire.ID) bool {
	value, err := pureType(graph, id)
	return err == nil && value.name == "i64"
}
func isIterationRead(graph wire.Envelope, id, binding wire.ID) bool {
	entity := graph.Entities[id]
	value, err := field(entity, 0x9f60)
	return entity.Schema == identity(0x90f6) && err == nil && value.Reference == binding
}
